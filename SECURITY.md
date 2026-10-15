# Security

## Reporting a vulnerability

Please don't open a public issue. Report it privately through GitHub: on the repository's **Security** tab, choose **Report a vulnerability**. You'll get a reply within a week, and a fix or a plan within 30 days of confirming the problem.

Report anything that could leak a key or change files it shouldn't, for example:

- a code path that prints, logs or writes a full RevenueCat API key (reports, previews and `rc-doctor facts` mask every key);
- `review --fix` or `migrate --apply` writing to a file or line other than the one the preview showed;
- a way for repository content (a config file, a crafted source file) to make rc-doctor run a command or read files outside the checked directory.

## Supported versions

Fixes go into the latest release.

## What rc-doctor does and doesn't touch

- It never handles RevenueCat credentials. Project data comes from RevenueCat's own `rc` CLI, which keeps its login and `RC_API_KEY` to itself.
- It runs `rc` and nothing else. It reads files under the directory it checks, plus a config or snapshot file you name, and doesn't follow symbolic links.
- Only `internal/fix` writes files, and only when the text it expects is still on the line it previewed.
- `rc-doctor snapshot` saves the project's public SDK keys, which ship inside apps by design. It never contains a secret key.
