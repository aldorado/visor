---
name: email-send
description: Use when asked to send an email, reply to an email, forward an email, or compose a message to send via email.
user-invocable: true
---

# Email Send Skill

Send emails using himalaya CLI.

## Sending a new email

Use himalaya's write command with MML (MIME Meta Language) format:

```bash
himalaya message write <<'EOF'
From: friday@sibwaxer.com
To: recipient@example.com
Subject: Your subject here

Your message body here.
EOF
```

## Replying to an email

```bash
himalaya message reply <id> <<'EOF'

Your reply text here.
EOF
```

## Forwarding an email

```bash
himalaya message forward <id> <<'EOF'
To: recipient@example.com

Optional note above the forwarded message.
EOF
```

## Notes

- Always confirm with the user before sending (show them the draft)
- From address is always friday@sibwaxer.com
- For replies, the original message is automatically included
- Keep the tone matching whatever the user asks for
