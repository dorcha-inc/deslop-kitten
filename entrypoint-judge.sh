#!/bin/sh
# Starts the llama.cpp server on localhost, then runs the scorer with the
# judge pointed at it. The scorer waits for the model to load before its
# first request.
set -eu
unset LLAMA_ARG_HOST

/app/llama-server \
  --model /model.gguf \
  --host 127.0.0.1 \
  --port 8080 \
  --ctx-size 8192 \
  --threads "$(nproc)" \
  --log-disable &

exec /vetkitten "$@" --judge-url http://127.0.0.1:8080/v1 --judge-model local
