---
name: scheduling
description: Use when asked to set a reminder, schedule something, create a recurring task, check scheduled tasks, or anything involving "remind me", "schedule", "every day at", "in 2 hours", "tomorrow at", "on feb 1st".
user-invocable: false
---

# Scheduling Skill

This is your reference for creating and managing scheduled tasks. You MUST actually create the cron job when asked to set a reminder - don't just acknowledge the request.

## How to Schedule

Use the CronManager class from TypeScript:

```bash
npx tsx -e "
import { CronManager } from './src/cron.ts';

const cron = new CronManager();
cron.addTask({
  name: 'unique-task-name',           // lowercase, hyphens
  schedule: '0 9 * * *',              // cron expression
  taskDescription: 'what to do',      // this becomes the agent prompt
  oneShot: false,                     // true = runs once then deletes itself
});
"
```

## Cron Schedule Format (node-cron)

```
┌───────────── minute (0-59)
│ ┌───────────── hour (0-23)
│ │ ┌───────────── day of month (1-31)
│ │ │ ┌───────────── month (1-12)
│ │ │ │ ┌───────────── day of week (0-6, 0=Sunday)
│ ││ │ │
* * * * *
```

### Common Patterns

| Want | Cron Expression |
|------|-----------------|
| Every day at 8am | `0 8 * * *` |
| Every hour | `0 * * * *` |
| Every 2 hours | `0 */2 * * *` |
| Weekdays at 9am | `0 9 * * 1-5` |
| Jan 29 at 5pm | `0 17 29 1 *` |
| Feb 1 at 10am | `0 10 1 2 *` |
| Every Monday at 10am | `0 10 * * 1` |
| First of month at noon | `0 12 1 * *` |

## One-Shot vs Recurring

- `oneShot: true`: for reminders ("remind me tomorrow at 5pm") - runs once, deletes itself
- `oneShot: false`: for recurring tasks ("every morning at 8am") - keeps running

## Task Description

The taskDescription becomes the prompt for a fresh agent session. Be specific:

```typescript
// good - specific action
taskDescription: "Send the user a reminder about buying toothfloss and shower gel"

// bad - vague
taskDescription: "Reminder"
```

## Managing Tasks

```bash
npx tsx -e "
import { CronManager } from './src/cron.ts';

const cron = new CronManager();

// list all scheduled tasks
const tasks = cron.listTasks();
console.log(JSON.stringify(tasks, null, 2));
"
```

```bash
npx tsx -e "
import { CronManager } from './src/cron.ts';

const cron = new CronManager();

// remove a task
cron.removeTask('task-name');
"
```

## Full Example: Setting a Reminder

User: "remind me tomorrow at 5pm to buy groceries"

```bash
npx tsx -e "
import { CronManager } from './src/cron.ts';

const tomorrow = new Date();
tomorrow.setDate(tomorrow.getDate() + 1);
const schedule = \`0 17 \${tomorrow.getDate()} \${tomorrow.getMonth() + 1} *\`;

const cron = new CronManager();
cron.addTask({
  name: 'reminder-groceries',
  schedule,
  taskDescription: 'Send the user a reminder to buy groceries',
  oneShot: true,
});
"
```

## What Happens When Task Runs

1. node-cron triggers the task
2. The scheduled task runner launches the agent with the taskDescription as prompt
3. The agent executes and can send messages, update news.md, etc.
4. If oneShot is true, the cron entry is removed after running

## Important

- Always use `oneShot: true` for one-time reminders
- Always confirm to the user that the reminder is set (but don't mention "one-shot" or implementation details)
- Task names should be descriptive and unique (use hyphens, lowercase)
- Test the cron expression mentally before setting
