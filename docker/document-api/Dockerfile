# syntax=docker/dockerfile:1
FROM rapidaai/rapida-python:3.11

ENV HOME=/app
ENV APP_HOME=$HOME
WORKDIR $APP_HOME

RUN mkdir -p $APP_HOME $APP_HOME/env && chown -R rapida-app:rapida-app $APP_HOME

ENV PYTHONDONTWRITEBYTECODE 1
ENV PYTHONUNBUFFERED 1

# Copy requirements
USER root
COPY --chown=rapida-app:rapida-app api/document-api/requirements.txt requirements.txt

# Install Python dependencies
RUN --mount=type=cache,target=/root/.cache/pip \
    pip install --only-binary :all: pillow-heif 2>/dev/null || \
    pip install --no-binary :all: pillow-heif || \
    echo "WARNING: pillow-heif installation had issues"

RUN --mount=type=cache,target=/root/.cache/pip \
    pip install -r requirements.txt

# Switch to non-root user for app runtime
USER rapida-app

# Copy app code
COPY --chown=rapida-app:rapida-app api/document-api/app ./app
COPY --chown=rapida-app:rapida-app docker/document-api/config.yaml ./env/document.yaml

EXPOSE 9010

CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "9010"]
