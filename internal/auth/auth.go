package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/x-qdo/clickup-billing-report/internal/clickup"
	"github.com/x-qdo/clickup-billing-report/internal/config"
	"golang.org/x/oauth2"
)

const (
	clickUpAuthURL    = "https://app.clickup.com/api"
	clickUpTokenURL   = "https://api.clickup.com/api/v2/oauth/token"
	stateCookieName   = "clickup_auth_state"
	sessionCookieName = "reporter_session"
	// Scopes required by the application
	requiredScopes = "" // ClickUp uses implicit scopes based on App settings
)

// Authenticator handles the OAuth2 flow with ClickUp.
type Authenticator struct {
	oauthConf *oauth2.Config
	store     config.DataStore // Use interface from config package
	log       *logrus.Entry
	appConfig *config.AppConfig
}

// NewAuthenticator creates a new Authenticator.
func NewAuthenticator(appConf *config.AppConfig) *Authenticator {
	logEntry := appConf.Logger.WithField("component", "Authenticator")

	// Construct the redirect URL robustly
	baseURL, err := url.Parse(appConf.Global.BaseURL)
	if err != nil {
		logEntry.WithError(err).Fatal("Invalid BASE_URL provided in config") // Fatal as it's critical
	}
	callbackPath, _ := url.Parse("/auth/callback") // Relative path for callback
	redirectURL := baseURL.ResolveReference(callbackPath).String()

	conf := &oauth2.Config{
		ClientID:     appConf.Global.ClickUpClientID,
		ClientSecret: appConf.Global.ClickUpClientSecret,
		// Scopes are implicit in ClickUp OAuth2 based on app settings
		Endpoint: oauth2.Endpoint{
			AuthURL:  clickUpAuthURL, // Base URL for auth initiation
			TokenURL: clickUpTokenURL,
		},
		RedirectURL: redirectURL,
	}

	logEntry.WithFields(logrus.Fields{
		"client_id":    conf.ClientID,
		"redirect_url": conf.RedirectURL,
		"auth_url":     conf.Endpoint.AuthURL,
		"token_url":    conf.Endpoint.TokenURL,
	}).Info("OAuth2 Config Initialized")

	return &Authenticator{
		oauthConf: conf,
		store:     appConf.Store, // Get store from AppConfig
		log:       logEntry,
		appConfig: appConf,
	}
}

// StartAuth generates the ClickUp authorization URL and state cookie.
func (a *Authenticator) StartAuth(w http.ResponseWriter, r *http.Request) string {
	// Generate random state
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		a.log.WithError(err).Error("Failed to generate random state")
		// Handle error appropriately (e.g., return error page URL)
		return "/error?msg=state_generation_failed" // Example error redirect
	}
	state := base64.URLEncoding.EncodeToString(b)

	// Determine if Secure flag should be set based on request or config
	// In Lambda behind API Gateway, check X-Forwarded-Proto header
	isSecure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"

	// Set state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    state,
		Expires:  time.Now().Add(10 * time.Minute), // Short expiry for state
		HttpOnly: true,
		Secure:   isSecure,
		Path:     "/", // Cookie accessible for all paths
		SameSite: http.SameSiteLaxMode,
	})

	// Construct the Auth URL manually as ClickUp expects specific query params
	// The oauth2 library's AuthCodeURL doesn't perfectly match ClickUp's requirement.
	authURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code&state=%s",
		a.oauthConf.Endpoint.AuthURL,
		url.QueryEscape(a.oauthConf.ClientID),
		url.QueryEscape(a.oauthConf.RedirectURL),
		url.QueryEscape(state),
	)
	// Note: ClickUp doesn't seem to require 'scope' in the auth URL itself.

	a.log.WithField("auth_url", authURL).Info("Generated ClickUp Auth URL")
	return authURL
}

// HandleCallback exchanges the code for a token and creates a user session.
func (a *Authenticator) HandleCallback(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	a.log.Info("Handling OAuth callback")

	// Check state cookie
	stateCookie, err := r.Cookie(stateCookieName)
	if err != nil {
		a.log.Warn("State cookie not found or error reading it")
		return fmt.Errorf("missing or invalid state cookie: %w", err)
	}
	// Clear the state cookie immediately
	isSecure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
	http.SetCookie(w, &http.Cookie{Name: stateCookieName, MaxAge: -1, Path: "/", Secure: isSecure, HttpOnly: true})

	queryState := r.URL.Query().Get("state")
	if queryState == "" || queryState != stateCookie.Value {
		a.log.WithFields(logrus.Fields{
			"query_state":  queryState,
			"cookie_state": stateCookie.Value,
		}).Error("State mismatch")
		return fmt.Errorf("invalid oauth state")
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		a.log.Error("No code received in callback")
		// Check for error query parameters from ClickUp
		clickupErr := r.URL.Query().Get("error")
		if clickupErr != "" {
			a.log.Errorf("Received error from ClickUp during auth: %s", clickupErr)
			return fmt.Errorf("clickup auth error: %s", clickupErr)
		}
		return fmt.Errorf("no code in callback request")
	}

	a.log.Info("Exchanging code for token")
	// Use context with timeout for token exchange
	exchangeCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	token, err := a.oauthConf.Exchange(exchangeCtx, code)
	if err != nil {
		a.log.WithError(err).Error("Failed to exchange code for token")
		// Check if context timed out
		if exchangeCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("timeout during code exchange: %w", err)
		}
		return fmt.Errorf("code exchange failed: %w", err)
	}

	if !token.Valid() {
		a.log.Error("Received invalid token")
		return fmt.Errorf("provider returned invalid token")
	}
	a.log.Info("Token received successfully")

	// --- Get User Info from ClickUp ---
	cuClient, err := clickup.GetAuthenticatedUserClient(ctx, a.oauthConf, token, a.appConfig.Logger)
	if err != nil {
		a.log.WithError(err).Error("Failed to create ClickUp client with new token")
		return fmt.Errorf("failed to initialize clickup client: %w", err)
	}

	userInfo, err := cuClient.GetUser(ctx)
	if err != nil {
		a.log.WithError(err).Error("Failed to get user info from ClickUp")
		return fmt.Errorf("failed to fetch user info: %w", err)
	}
	a.log.WithFields(logrus.Fields{
		"user_id":   userInfo.ID,
		"user_name": userInfo.Username,
		"email":     userInfo.Email,
	}).Info("Fetched user info from ClickUp")
	// ---------------------------------

	// --- Create Session ---
	sessionID := uuid.NewString()

	// Encrypt token.AccessToken using SessionSecret
	encryptedToken, err := encrypt(token.AccessToken, a.appConfig.Global.SessionSecret)
	if err != nil {
		a.log.WithError(err).Error("Failed to encrypt access token for session")
		return fmt.Errorf("session token encryption failed: %w", err)
	}
	a.log.Debug("Access token encrypted")

	sessionExpiry := token.Expiry.UTC()
	if sessionExpiry.IsZero() {
		sessionExpiry = time.Now().AddDate(1, 0, 0).UTC() // 1 year from now
		a.log.Info("Token has no expiry (zero time), setting session expiry to 1 year")
	}

	session := config.SessionState{
		SessionID:    sessionID,
		ClickUpToken: encryptedToken,
		UserID:       strconv.Itoa(userInfo.ID),
		UserName:     userInfo.Username,
		ExpiresAt:    sessionExpiry,
		TTL:          sessionExpiry.UTC().Unix(),
	}

	err = a.store.SaveSession(ctx, session)
	if err != nil {
		a.log.WithError(err).Error("Failed to save session to store")
		return fmt.Errorf("failed to save session: %w", err)
	}
	a.log.WithField("user_id", session.UserID).Info("Session saved")

	// Set session cookie
	// Set session cookie using the calculated expiry
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Expires:  sessionExpiry, // Use calculated expiry
		HttpOnly: true,
		Secure:   isSecure,
		Path:     "/", // Make cookie available for all paths
		SameSite: http.SameSiteLaxMode,
	})
	a.log.Info("Session cookie set")

	return nil // Success
}

// GetSessionFromRequest retrieves the user session based on the session cookie.
// It returns the session state and the decrypted access token.
func (a *Authenticator) GetSessionFromRequest(r *http.Request) (*config.SessionState, string, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		if err == http.ErrNoCookie {
			return nil, "", nil // No session cookie, user not logged in
		}
		a.log.WithError(err).Warn("Error reading session cookie")
		return nil, "", fmt.Errorf("invalid session cookie")
	}

	sessionID := cookie.Value
	if sessionID == "" {
		return nil, "", nil // Empty cookie value
	}

	session, err := a.store.GetSession(r.Context(), sessionID)
	if err != nil {
		if err == config.ErrNotFound { // Use error from config package
			a.log.WithField("session_id_prefix", sessionID[:min(len(sessionID), 8)]).Warn("Session ID not found in store or expired")
			return nil, "", nil // Session not found or expired
		}
		a.log.WithError(err).Error("Failed to retrieve session from store")
		return nil, "", fmt.Errorf("failed to retrieve session")
	}

	// Decrypt session.ClickUpToken
	decryptedToken, err := decrypt(session.ClickUpToken, a.appConfig.Global.SessionSecret)
	if err != nil {
		a.log.WithError(err).WithField("user_id", session.UserID).Error("Failed to decrypt session token")
		// Consider deleting the invalid session?
		// a.store.DeleteSession(r.Context(), sessionID)
		return nil, "", fmt.Errorf("session token decryption failed")
	}
	a.log.WithField("user_id", session.UserID).Debug("Session token decrypted successfully")

	return session, decryptedToken, nil
}

// Logout deletes the user session and clears the cookie.
func (a *Authenticator) Logout(w http.ResponseWriter, r *http.Request) error {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil && cookie.Value != "" {
		// Delete from store first
		delErr := a.store.DeleteSession(r.Context(), cookie.Value)
		if delErr != nil && delErr != config.ErrNotFound { // Use error from config package
			// Log error but proceed to clear cookie anyway
			a.log.WithError(delErr).WithField("session_id_prefix", cookie.Value[:min(len(cookie.Value), 8)]).Error("Failed to delete session from store during logout")
		} else if delErr == nil {
			a.log.Info("Session deleted from store during logout")
		}
	} else if err != http.ErrNoCookie {
		a.log.WithError(err).Warn("Error reading session cookie during logout")
	}

	// Clear the session cookie regardless of store deletion status
	isSecure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		MaxAge:   -1, // Delete cookie
		HttpOnly: true,
		Secure:   isSecure,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})
	a.log.Info("User logged out, session cookie cleared")
	return nil
}
