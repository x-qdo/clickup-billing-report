# ClickUp Reporter Frontend

A modern React-based wizard interface for generating ClickUp time tracking and billable reports. Built with TypeScript, Tailwind CSS, and Vite for a streamlined development experience.

## Features

- **Wizard-Based Workflow**: Step-by-step guided process for report generation
- **Time Tracking Reports**: Generate monthly reports with developer coefficients
- **Billable Reports**: Create client-specific billing reports
- **State Persistence**: Wizard progress saved to localStorage
- **Excel Downloads**: Export reports in professional Excel format
- **Responsive Design**: Works on desktop and mobile devices
- **Modern UI**: Clean, accessible interface with Tailwind CSS

## Prerequisites

- Node.js 18+ and npm
- Access to the ClickUp Reporter Go backend API

## Quick Start

1. **Install dependencies**:
   ```bash
   npm install
   ```

2. **Configure environment**:
   ```bash
   cp .env.example .env
   # Edit .env with your API configuration
   ```

3. **Start development server**:
   ```bash
   npm run dev
   ```

4. **Open your browser**:
   Navigate to `http://localhost:5173`

## Environment Configuration

Create a `.env` file in the frontend directory:

```env
# API Configuration
VITE_API_BASE_URL=http://localhost:8080

# Development Configuration
VITE_DEV_MODE=true
VITE_DEBUG=false
VITE_ENVIRONMENT=development
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `VITE_API_BASE_URL` | Backend API base URL | `http://localhost:8080` |
| `VITE_DEV_MODE` | Enable development features | `true` |
| `VITE_DEBUG` | Enable debug logging | `false` |
| `VITE_ENVIRONMENT` | Environment name for display | `development` |

## Available Scripts

| Command | Description |
|---------|-------------|
| `npm run dev` | Start development server with hot reload |
| `npm run build` | Build for production |
| `npm run preview` | Preview production build locally |
| `npm run lint` | Run ESLint |

## Project Structure

```
src/
├── components/          # Reusable UI components
│   ├── Layout.tsx      # Main application layout
│   ├── ProtectedRoute.tsx  # Authentication guard
│   └── NotificationList.tsx  # Toast notifications
├── contexts/           # React contexts
│   ├── AuthContext.tsx # Authentication state
│   └── NotificationContext.tsx  # Notification system
├── pages/              # Main application pages
│   ├── Dashboard.tsx   # Home dashboard
│   ├── Login.tsx       # Authentication page
│   ├── Settings.tsx    # Configuration management
│   ├── TimeTrackingWizard.tsx  # Time tracking workflow
│   └── BillableWizard.tsx      # Billable reporting workflow
├── services/           # API and external services
│   └── api.ts         # API client with authentication
├── App.tsx            # Main application component
└── main.tsx           # Application entry point
```

## Wizard Workflows

### Time Tracking Report Wizard

A 5-step process for generating time tracking reports:

1. **Select Report Month** - Choose the month for report generation
2. **Generate Initial Report** - Create report without updating ClickUp
3. **Review & Verify** - Examine calculations and download Excel
4. **Update ClickUp Fields** - Generate final report with field updates
5. **Complete** - Workflow finished with download options

### Billable Report Wizard

A 5-step process for client billing:

1. **Select Client** - Choose client from configured list
2. **Generate Billable Report** - Create report without updates
3. **Review Billable Tasks** - Verify tasks and totals
4. **Mark as Invoiced** - Update InvoicedHours in ClickUp
5. **Complete** - Workflow finished with final reports

## State Management

- **Authentication**: Managed via React Context with persistent sessions
- **Wizard Progress**: Automatically saved to localStorage for recovery
- **Notifications**: Toast-style notifications for user feedback

### Wizard State Persistence

Both wizards automatically save progress to localStorage:

```typescript
// Saved state includes:
{
  currentStep: number,
  reportDate: string,        // Time tracking wizard
  clientName: string,        // Billable wizard
  initialReportGenerated: boolean,
  finalReportGenerated: boolean,
  reportData: object | null
}
```

## API Integration

The frontend communicates with the Go backend through a REST API:

### Authentication Endpoints
- `GET /auth/me` - Check authentication status
- `POST /auth/logout` - Sign out user
- `GET /auth/clickup` - Initiate ClickUp OAuth flow

### Report Endpoints
- `POST /report/timetrack` - Generate time tracking reports
- `POST /report/billable` - Generate billable reports

### Request Format

All report requests use form-encoded data:

```typescript
// Time tracking example
const formData = new URLSearchParams();
formData.append('report_date', '2024-01');
formData.append('refresh_billable', 'on');  // Optional
formData.append('format', 'excel');         // Optional

// Billable report example
const formData = new URLSearchParams();
formData.append('client_name', 'Acme Corp');
formData.append('refresh_invoiced', 'on');  // Optional
formData.append('format', 'excel');         // Optional
```

## Styling and Components

### Tailwind CSS

The project uses Tailwind CSS for styling with custom components:

```css
/* Custom component classes */
.btn-primary { /* Blue primary button */ }
.btn-secondary { /* Gray secondary button */ }
.form-input { /* Standard form input */ }
.form-select { /* Standard form select */ }
```

### Icons

Uses Heroicons for consistent iconography:

```typescript
import { ClockIcon, CurrencyDollarIcon } from '@heroicons/react/24/outline';
```

## Development Guidelines

### Code Style

- Use TypeScript for all components
- Follow React functional component patterns
- Use custom hooks for complex state logic
- Prefer composition over inheritance

### Error Handling

- Use try/catch blocks for async operations
- Display user-friendly error messages
- Log detailed errors to console in development
- Handle network failures gracefully

### Accessibility

- Use semantic HTML elements
- Provide appropriate ARIA labels
- Ensure keyboard navigation works
- Maintain color contrast ratios

## Building for Production

1. **Build the application**:
   ```bash
   npm run build
   ```

2. **Preview locally**:
   ```bash
   npm run preview
   ```

3. **Deploy**:
   Upload the `dist/` directory to your web server or CDN.

### Environment Variables for Production

```env
VITE_API_BASE_URL=https://your-api-gateway-url.amazonaws.com
VITE_DEV_MODE=false
VITE_ENVIRONMENT=production
```

## Browser Support

- Chrome 88+
- Firefox 85+
- Safari 14+
- Edge 88+

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Workflow

1. **Start backend**: Ensure the Go API is running on port 8080
2. **Start frontend**: Run `npm run dev`
3. **Test authentication**: Visit `/auth/clickup` to test OAuth flow
4. **Test wizards**: Use the step-by-step workflows
5. **Check responsive design**: Test on different screen sizes

## Troubleshooting

### Common Issues

**Authentication fails**: 
- Verify backend is running and accessible
- Check `VITE_API_BASE_URL` in `.env`
- Ensure ClickUp OAuth is configured correctly

**Wizard state lost**:
- Check browser localStorage
- Verify no errors in browser console
- Clear localStorage and restart if needed

**Excel downloads fail**:
- Check network tab for API errors
- Verify backend can generate Excel files
- Ensure proper content-type headers

**Build failures**:
- Clear `node_modules` and reinstall
- Check for TypeScript errors
- Verify all environment variables are set

## License

This project is licensed under the MIT License - see the LICENSE file for details.