gcloud functions deploy intercom-webhook-converter --runtime go111 --entry-point=WebHookConverter --env-vars-file .env.yaml --trigger-http
