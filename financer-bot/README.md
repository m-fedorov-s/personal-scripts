# Financer Bot

A Telegram bot for personal finance tracking, built with Go and BadgerDB.

## Features

- **Expense Tracking**: Easily log your daily expenses via simple text messages.
- **Balance Management**: Set and track your current balance and daily spending limits.
- **Automated Reports**: Receive daily financial summaries at your preferred time.
- **Statistics**:
  - Real-time balance updates.
  - Monthly daily spending averages.

## Commands
- `/setBalance <amount>`: Set current balance.
- `/setLimit <amount>`: Set monthly spending limit.
- `/setReportTime <HH:MM>`: Set preferred daily report time (e.g., 09:00).
- `/toggleReport`: Enable or disable daily subscription reports.
- `/report`: Get current financial report.

## Getting Started

### Prerequisites
- Go 1.21+
- Telegram Bot Token (from @BotFather)
- Docker

### Installation
1. Clone the repository.
2. Set your bot token: `export BOT_TOKEN=your_token_here`.
3. Run the bot: `docker compose up --build`.

## Architecture
The bot follows a clean separation of concerns:
- `internal/handler`: Telegram command and message handlers.
- `internal/storage`: Persistent storage layer with BadgerDB.
- `internal/report`: Financial statistics and report generation logic.
- `internal/worker`: Background scheduler for automated tasks.
