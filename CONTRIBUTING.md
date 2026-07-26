# Contributing to Aether

First off, thank you for considering contributing to Aether! It's people like you that make Aether such a great tool for the community.

## Core Philosophy

Aether is designed with a single core principle: **This launcher exists to launch Minecraft, and nothing more.**

Every feature that is not essential to launching the game is implemented as an extension. By keeping the core launcher intentionally minimal, Aether remains fast, predictable, and free from bloat. When contributing, please keep this philosophy in mind. If you are proposing a new feature, consider if it would be better implemented as an extension rather than in the core launcher.

## How to Contribute

### 1. Building an Extension
The best way to contribute to the Aether ecosystem is by building extensions! Check out the [Extensions Guide](docs/EXTENSIONS.md) and the [API Documentation](docs/API.md) to get started. You can use the Aether SDK and CLI to scaffold and package your extensions.

### 2. Reporting Bugs
If you find a bug, please search the issue tracker to see if it has already been reported. If not, open a new issue with a clear title and description, steps to reproduce, and any relevant logs or screenshots.

### 3. Suggesting Enhancements
Have an idea for a core feature or improvement? Open an issue and describe your idea. Please remember our core philosophy when suggesting features.

### 4. Code Contributions
1. Fork the repository.
2. Create a new branch (`git checkout -b feature/my-feature`).
3. Make your changes, following the [Styleguide](docs/STYLEGUIDE.md) and [Architecture](docs/ARCHITECTURE.md) guidelines.
4. Commit your changes (`git commit -m 'Add some feature'`).
5. Push to the branch (`git push origin feature/my-feature`).
6. Open a Pull Request.

### 5. Building from Source
To test your changes locally, you'll need to build the launcher from source.

**Prerequisites:**
- [Go 1.22+](https://go.dev/doc/install)
- [Node.js 20+](https://nodejs.org/)
- [Wails CLI v2](https://wails.io/docs/gettingstarted/installation)

**Development Commands:**
```bash
# Run the application in development mode with live reload
wails dev

# Build a production executable
wails build
```

## Code of Conduct

Please note that this project is released with a Contributor Code of Conduct. By participating in this project you agree to abide by its terms.
