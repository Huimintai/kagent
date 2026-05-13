# ui/ - Next.js Web Interface

Agent management UI built with Next.js App Router.

## Tech Stack

| Technology | Version | Role |
|-----------|---------|------|
| Next.js | 16.x | Framework (App Router) |
| React | 19.x | UI library |
| TypeScript | 5.9 | Language |
| Tailwind CSS | 3.x | Styling |
| Radix UI | latest | Accessible primitives |
| Zustand | 5.x | State management |
| Zod | 4.x | Schema validation |
| react-hook-form | 7.x | Form management |
| Lucide React | latest | Icons |

## Structure

| Directory | Purpose |
|-----------|---------|
| `src/` | Application source code |
| `cypress/` | E2E tests (Cypress) |
| `public/` | Static assets |
| `conf/` | Configuration files |
| `scripts/` | Build/utility scripts |

## Scripts

| Command | Action |
|---------|--------|
| `npm run dev` | Dev server (port 8001) |
| `npm run build` | Production build |
| `npm run lint` | ESLint |
| `npm run test` | Jest unit tests |
| `npm run test:vitest` | Vitest unit tests |
| `npm run test:e2e` | Cypress E2E (starts dev server) |
| `npm run storybook` | Storybook (port 6006) |

## Key Integrations

- OIDC authentication via jose JWT validation
- A2A SDK (`@a2a-js/sdk`) for agent-to-agent protocol
- Feature flags via config store
- Dark/light theme via next-themes
