# Security

Do not commit API keys, OAuth secrets, `.env` files, or local `.forage/state.db` files.

Forage stores credentials in the OS keychain by default. Environment variables are read as fallback/override and are never persisted into `.forage/config.yaml`.

Report security issues privately to the repository owner before filing public issues.
