---
name: email-read
description: Use when asked to check email, read emails, look at inbox, search emails, or anything involving reading/checking email messages.
user-invocable: true
---

# Email Read Skill

Check and read emails using himalaya CLI.

## Commands

List recent emails (inbox):
```bash
himalaya envelope list
```

List emails in a specific folder:
```bash
himalaya envelope list -f Sent
himalaya envelope list -f Drafts
```

Read a specific email by ID:
```bash
himalaya message read <id>
```

Search emails:
```bash
himalaya envelope list -q "subject:keyword"
himalaya envelope list -q "from:someone@example.com"
```

List with more results:
```bash
himalaya envelope list -s 20
```

## Notes

- Default account is "friday" (friday@sibwaxer.com)
- Email IDs are shown in the envelope list output
- Use `message read` to get the full body of an email
- Summarize emails naturally for the user, don't dump raw output
