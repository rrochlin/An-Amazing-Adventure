# An-Amazing-Adventure

### Notes on running
- you will need Go installed
- need to have access to the doppler project for secrets
- reccommended to use pnpm for client

### Running the project
- make sure to sign into doppler with `doppler login`
- setup doppler in server by running `doppler setup` inside of server folder
- start the server in a terminal with `doppler run -- go run .`
- in a seperate terminal do `doppler setup` in the client directory
- start the game with `doppler run -- pnpm dev`

### Deployment

CI/CD runs automatically on pushes to `main` via GitHub Actions. The workflow builds Docker images and deploys to Google Cloud Run.

**Troubleshooting failed deployments:**

If deployment fails with an error like:
```
Image 'mirror.gcr.io/rrochlin/an-amazing-adventure-client:sha-xxx' not found
```

This is a Docker Hub mirror sync delay - GCP pulls Docker Hub images through `mirror.gcr.io` which may not have new images immediately. The workflow retries automatically with exponential backoff. If all retries fail:
1. Wait a few minutes and re-run the GitHub Actions workflow manually
2. Check the Actions logs for specific error details
