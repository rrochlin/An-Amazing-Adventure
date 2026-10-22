#TerraformRepoIntegration
### The Two Repositories

**1. The Monorepo (per application)**
This is where you do your daily work.

```text
my-app/
├── client/
├── server/
└── .gitignore
```

The `server/` directory will also contain an `infra/` folder that is linked — via git subtree — to the infrastructure repo. From your perspective as a developer, it just looks like files sitting in your project.

**2. The Infrastructure Environments Repo (shared, one for all apps)**
This is the single Terraform repo you maintain. It references each app's infrastructure config by git tag. This is the only place Terraform pipelines run. Adding a new app is one block in a `main.tf`.

---

### How the Linking Works (Git Subtree)

The `server/infra/` folder in your monorepo is a subtree — a live copy of the app's infrastructure config. You can:

- Edit Terraform files directly inside your monorepo
- Push those changes back to a dedicated infra branch
- Open a PR from the command line with `gh pr create`

No submodule friction. Other developers just clone the monorepo and everything is there.

---

### The Deployment Flow

```
You edit server/infra/ in your monorepo
        ↓
git subtree push → creates a branch on the app infra repo
        ↓
gh pr create → opens PR for review
        ↓
PR merges → infra repo gets a new git tag
        ↓
Environment repo's main.tf is updated to point at new tag
        ↓
Terraform pipeline runs once, deploys the change
```

---

### Shared Infrastructure

Resources that multiple apps share (VPC, Event Bus, etc.) live in a separate `shared/` layer inside the environment repo. App modules never reference these directly — the values are passed in as variables. This keeps each app fully isolated.

---

### Onboarding a New App

1. Create the monorepo for the new app
2. Add a subtree link pointing to an app infra repo (can be templated)
3. Add one `module` block to the environment repo's `main.tf`
4. Terraform picks it up on the next run

No new pipelines. No new CI config.

---

### What Lives Where

| Thing                | Lives In                           |
| -------------------- | ---------------------------------- |
| Frontend code        | Monorepo `client/`                 |
| Backend code         | Monorepo `server/`                 |
| App Terraform config | Monorepo `server/infra/` (subtree) |
| Terraform pipeline   | Environment repo only              |
| Shared AWS resources | Environment repo `shared/`         |
| Deployment triggers  | Environment repo via git tags      |