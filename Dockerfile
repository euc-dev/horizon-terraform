#############################################################################
# build markdown to html

# get python image to create static pages
FROM python:alpine AS builder

# install require pyton packages (mkdocs, etc)
WORKDIR /workspace
COPY requirements.txt .
RUN apk add --no-cache git && \
    pip install --no-cache-dir --upgrade pip && \
    pip install --no-cache-dir -r requirements.txt

# build the static pages
COPY . .
RUN mkdocs build -f mkdocs-local.yml

#############################################################################
# build the final image with content from builder

# get base image
FROM httpd:2.4-alpine AS final

WORKDIR /usr/local/apache2/htdocs/

# add metadata via labels
LABEL com.omnissa.version="0.0.1"
LABEL com.omnissa.git.repo="https://github.com/euc-dev/horizon-terraform"
LABEL com.omnissa.git.commit="DEADBEEF"
LABEL com.omnissa.maintainer.name="Richard Croft"
LABEL com.omnissa.maintainer.email="rcroft@broadcom.com"
LABEL com.omnissa.released="9999-99-99"
LABEL com.omnissa.based-on="httpd:2.4-alpine"
LABEL com.omnissa.project="horizon-terraform"

# copy the html to wwwroot
#COPY --chmod=nobody:nogroup --from=builder /app/html ./
COPY --from=builder /workspace/.site ./

#############################################################################
# vim: ft=unix sync=dockerfile ts=4 sw=4 et tw=78:
