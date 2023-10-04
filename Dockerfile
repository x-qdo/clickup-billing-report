FROM public.ecr.aws/docker/library/python:3.12-bullseye

ENV POETRY_VERSION='1.2.2' \
    POETRY_HOME=/etc/poetry \
    PATH="/etc/poetry/bin:${PATH}" \
    POETRY_VIRTUALENVS_CREATE=false
WORKDIR /app

RUN apt-get update -qq;\
    apt-get install -yq libxml2 libxslt1.1 libffi7 libssl1.1 python3-cryptography  &&\
    apt-get clean -yqq &&\
    rm -rf /var/lib/apt/lists/*

RUN curl -sSL https://install.python-poetry.org | python3 - && \
    poetry --version

COPY --chown=nobody:nogroup pyproject.toml poetry.lock ./
RUN poetry install --no-root --no-dev

COPY --chown=nobody:nogroup . ./

CMD [ "poetry", "run", "gunicorn", "--conf", "gunicorn_conf.py", "--bind", "0.0.0.0:8080", "app:app"]

# Nobody
USER 65534
