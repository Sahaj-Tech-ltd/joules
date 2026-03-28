# Email Setup

Email is optional. Without it, Joules still works fine — verification codes are printed to Docker logs instead of sent by email.

If you want users to receive proper email verification and (in the future) password reset emails, configure SMTP.

---

## Without Email (Default)

After signing up, find your verification code in the logs:

```bash
docker compose logs joule | grep -i verification
```

You'll see something like:
```
joule  | Verification code for user@example.com: 847291
```

Enter this code on the verification page to activate your account.

---

## Configuring SMTP

Add these variables to your `.env`:

```env
SMTP_HOST=your-smtp-server
SMTP_PORT=465
SMTP_USER=your@email.com
SMTP_PASS=your-password-or-app-password
```

### Port 465 vs 587

- **465** — Implicit TLS. The connection is encrypted from the start. Recommended.
- **587** — STARTTLS. Starts unencrypted, then upgrades. Used by some providers.

Use whatever your provider specifies.

---

## Provider-Specific Examples

### Gmail

Gmail requires an **App Password** — you cannot use your regular Gmail password.

1. Enable 2-Factor Authentication on your Google account
2. Go to [myaccount.google.com/apppasswords](https://myaccount.google.com/apppasswords)
3. Create an app password for "Mail"

```env
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=you@gmail.com
SMTP_PASS=xxxx xxxx xxxx xxxx    # The 16-character app password
```

### Fastmail

```env
SMTP_HOST=smtp.fastmail.com
SMTP_PORT=465
SMTP_USER=you@fastmail.com
SMTP_PASS=your-app-password
```

### Mailgun

```env
SMTP_HOST=smtp.mailgun.org
SMTP_PORT=587
SMTP_USER=postmaster@mg.yourdomain.com
SMTP_PASS=your-mailgun-smtp-password
```

### Self-Hosted (e.g. Maddy, Stalwart)

```env
SMTP_HOST=mail.yourdomain.com
SMTP_PORT=465
SMTP_USER=hello@yourdomain.com
SMTP_PASS=your-password
```

---

## Testing Your SMTP Config

After updating `.env`, restart Joules and create a new account (or a test account) to trigger a verification email:

```bash
docker compose down
docker compose up -d
docker compose logs -f joule
```

If there's a problem, you'll see an SMTP error in the logs. Common issues:

| Error | Fix |
|-------|-----|
| `connection refused` | Wrong host or port |
| `authentication failed` | Wrong username/password; use an app password for Gmail |
| `certificate error` | Try port 587 instead of 465, or vice versa |
