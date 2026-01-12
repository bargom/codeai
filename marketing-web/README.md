# CodeAI Marketing Website

Static marketing website for the CodeAI project.

## Development

### Install Dependencies
```bash
npm install
```

### Local Development
```bash
npm run dev
```

### Build for Production
```bash
npm run build
```

### Test Locally (Static Files)
```bash
# From the marketing-web directory
python -m http.server 8000

# Or serve the dist folder
cd dist && python -m http.server 8000
```

Then open http://localhost:8000 in your browser.

## GitLab Pages Deployment

This site is automatically deployed to GitLab Pages when changes are pushed to the `main` branch.

### How It Works

1. When code is pushed to `main`, GitLab CI runs the `pages` job
2. The job copies contents of `marketing-web/` to the `public/` directory
3. GitLab Pages serves the `public/` directory as a static website

### URL Format

After deployment, the site will be available at:
```
https://<username>.gitlab.io/codeai
```

Or for group projects:
```
https://<group>.gitlab.io/codeai
```

### Configuration

The deployment is configured in `.gitlab-ci.yml` at the repository root:
```yaml
pages:
  stage: deploy
  script:
    - mkdir -p public
    - cp -r marketing-web/* public/
  artifacts:
    paths:
      - public
  only:
    - main
```

### Troubleshooting

- **404 errors**: Ensure the `index.html` exists in the root of `marketing-web/`
- **CSS/JS not loading**: Check that asset paths are relative, not absolute
- **Build not triggering**: Verify changes are pushed to the `main` branch
