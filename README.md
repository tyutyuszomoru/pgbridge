# PostgreSQL Bridge (pgbridge)

A daemon service that acts as a bridge between PostgreSQL databases and external systems, using PostgreSQL's LISTEN/NOTIFY mechanism for asynchronous event processing.

## Table of Contents
- [Overview](#overview)
- [Features](#features)
- [Database Prerequisites](#database-prerequisites)
- [Database User Setup](#database-user-setup)
- [Module-Specific Permissions](#module-specific-permissions)
- [Configuration](#configuration)
- [Installation](#installation)
- [Running Tests](#running-tests)

## Overview

pgbridge connects to one or more PostgreSQL databases and listens for NOTIFY events on specific channels. When events are received, the appropriate module processes them (sending emails, executing async SQL, processing files, etc.).

## Features

- Multi-database support with independent connection pools
- Automatic reconnection with exponential backoff
- Health monitoring for all connections
- Modular architecture for easy extension
- Comprehensive logging to both system logs and database tables

## Database Prerequisites

Before adding a database to pgbridge configuration, you must:

1. Create a dedicated PostgreSQL role/user for pgbridge
2. Grant appropriate permissions based on which modules you'll use
3. Ensure the database accepts connections from the pgbridge host

## Database User Setup

### 1. Create the pgbridge Role

It's recommended to create a dedicated role for pgbridge with a strong password:

```sql
-- Create the pgbridge role
CREATE ROLE pgb WITH LOGIN PASSWORD 'your_secure_password_here';

-- Optional: Add a comment for documentation
COMMENT ON ROLE pgb IS 'Dedicated role for pgbridge daemon service';
```

### 2. Grant Basic Permissions

All pgbridge installations require these base permissions:

```sql
-- Connect to your target database first
\c your_database_name

-- Grant connection privileges
GRANT CONNECT ON DATABASE your_database_name TO pgb;

-- Grant schema creation (pgbridge creates a 'pgb' schema for logging)
GRANT CREATE ON DATABASE your_database_name TO pgb;

-- After pgbridge creates the pgb schema on first run, grant usage
-- (You can run this preemptively or after first initialization)
GRANT USAGE ON SCHEMA pgb TO pgb;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA pgb TO pgb;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA pgb TO pgb;

-- Ensure future tables in pgb schema are accessible
ALTER DEFAULT PRIVILEGES IN SCHEMA pgb
    GRANT ALL PRIVILEGES ON TABLES TO pgb;
ALTER DEFAULT PRIVILEGES IN SCHEMA pgb
    GRANT ALL PRIVILEGES ON SEQUENCES TO pgb;
```

### 3. Verify Basic Setup

Test the connection and permissions:

```sql
-- Connect as the pgb user to verify
\c your_database_name pgb

-- You should be able to create the schema
CREATE SCHEMA IF NOT EXISTS pgb;

-- Verify you can create tables in the schema
CREATE TABLE IF NOT EXISTS pgb.test (id serial);
DROP TABLE pgb.test;
```

## Module-Specific Permissions

Depending on which modules you enable in your configuration, additional permissions are required:

### pgb_mail Module

The mail module reads from `pgb.pgb_mail` and `pgb.pgb_mail_settings` tables.

**Permissions needed:**
```sql
-- Grant usage on the public schema (or wherever your application tables are)
GRANT USAGE ON SCHEMA public TO pgb;

-- pgbridge will create pgb.pgb_mail and pgb.pgb_mail_settings tables
-- No additional permissions needed beyond base setup
```

**Note:** The application that triggers emails must be able to write to `pgb.pgb_mail` and execute:
```sql
NOTIFY pgb_mail, 'mail_id';
```

### pgb_notify Module

The notify module reads notifications from one database and forwards them to a central database.

**Permissions needed on source database:**
```sql
-- pgbridge will create pgb.pgb_notify table
-- No additional permissions needed beyond base setup
```

**Permissions needed on central/target database:**
```sql
-- The pgb user needs INSERT permissions on the target notification table
GRANT USAGE ON SCHEMA public TO pgb;  -- or your target schema
GRANT INSERT ON public.central_notifications TO pgb;  -- adjust table name
GRANT USAGE ON SEQUENCE public.central_notifications_id_seq TO pgb;  -- if using serial
```

### pgb_async Module ⚠️ CRITICAL SECURITY CONSIDERATION

The async module executes arbitrary SQL statements. This is powerful but potentially dangerous.

**⚠️ IMPORTANT SECURITY WARNINGS:**
- The `pgb` user will execute any SQL statement written to `pgb.pgb_async`
- Grant ONLY the minimum permissions needed for your use case
- Consider using a separate restricted role for async operations
- **NEVER** grant superuser or dangerous permissions like `DROP DATABASE`
- Implement application-level validation before writing to `pgb.pgb_async`

**Minimum permissions for pgb_async:**
```sql
-- For SELECT-only async operations (safest)
GRANT USAGE ON SCHEMA public TO pgb;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO pgb;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT ON TABLES TO pgb;
```

**For async operations that modify data:**
```sql
-- Grant permissions on specific tables only
GRANT USAGE ON SCHEMA public TO pgb;

-- Example: Grant permissions on specific tables
GRANT SELECT, INSERT, UPDATE, DELETE ON public.orders TO pgb;
GRANT SELECT, INSERT, UPDATE, DELETE ON public.order_items TO pgb;
GRANT SELECT, UPDATE ON public.inventory TO pgb;

-- Grant sequence usage for INSERT operations
GRANT USAGE ON SEQUENCE public.orders_id_seq TO pgb;
GRANT USAGE ON SEQUENCE public.order_items_id_seq TO pgb;

-- For future tables (if needed)
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO pgb;
```

**For async operations requiring DDL (use with extreme caution):**
```sql
-- Only if you need CREATE/ALTER/DROP operations
GRANT CREATE ON SCHEMA public TO pgb;

-- Grant specific DDL permissions (PostgreSQL 14+)
-- Note: Be very careful with these permissions
GRANT ALL PRIVILEGES ON SCHEMA public TO pgb;
```

**Recommended: Use a separate restricted role for async:**
```sql
-- Create a more restricted role for async operations
CREATE ROLE pgb_async WITH LOGIN PASSWORD 'different_secure_password';

-- Grant only what's needed
GRANT USAGE ON SCHEMA public TO pgb_async;
GRANT SELECT, INSERT, UPDATE ON specific_table TO pgb_async;

-- Configure pgbridge to use pgb_async role for async module
-- (This would require code changes to support per-module credentials)
```

### pgb_calendar Module

The calendar module processes calendar/scheduling events.

**Permissions needed:**
```sql
-- pgbridge will create pgb.pgb_calendar table
-- Grant access to any tables the calendar needs to read/write
GRANT USAGE ON SCHEMA public TO pgb;
GRANT SELECT ON public.events TO pgb;  -- adjust as needed
GRANT SELECT, INSERT, UPDATE ON public.reminders TO pgb;  -- adjust as needed
```

### pgb_file Module

The file module executes filesystem operations based on database records.

**Permissions needed:**
```sql
-- No special database permissions beyond base setup
-- File system permissions are handled at the OS level
```

**OS-level considerations:**
- The system user running pgbridge needs read/write/execute permissions on target directories
- Consider running pgbridge with a dedicated system user with restricted filesystem access

### pgb_csv Module

The CSV module imports CSV files into database tables.

**Permissions needed:**
```sql
-- Grant CREATE permission if auto-creating tables
GRANT CREATE ON SCHEMA public TO pgb;

-- Grant INSERT on existing tables
GRANT USAGE ON SCHEMA public TO pgb;
GRANT INSERT ON ALL TABLES IN SCHEMA public TO pgb;

-- For auto-created tables
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT INSERT ON TABLES TO pgb;
```

### pgb_xls Module

The Excel module processes Excel files using Python scripts.

**Permissions needed:**
```sql
-- Depends on what the Python scripts do
-- Minimum: ability to read file metadata
GRANT USAGE ON SCHEMA pgb TO pgb;

-- If scripts write back to database:
GRANT USAGE ON SCHEMA public TO pgb;
GRANT INSERT, UPDATE ON specific_tables TO pgb;
```

## Module Usage Examples

This section provides practical examples of how to use each module from within your PostgreSQL database.

### Using pgb_mail Module

The `pgb_mail` module allows you to send emails asynchronously from PostgreSQL triggers, functions, or direct SQL.

**1. Setup SMTP Configuration:**

```sql
-- Insert your SMTP server settings
INSERT INTO pgb.pgb_mail_settings (
    smtp_server,
    smtp_port,
    is_tls,
    is_ssl,
    smtp_user,
    smtp_password
) VALUES (
    'smtp.gmail.com',        -- SMTP server
    587,                      -- Port (587 for TLS, 465 for SSL, 25 for plain)
    true,                     -- Use STARTTLS
    false,                    -- SSL from start (false if using TLS)
    'your-email@gmail.com',   -- SMTP username
    'your-app-password'       -- SMTP password or app-specific password
) RETURNING id;

-- Store the returned ID for use in mail records
```

**2. Send an Email:**

```sql
-- Insert a mail record
INSERT INTO pgb.pgb_mail (
    mail_setting_id,
    header_from,
    header_to,
    header_cc,
    header_bcc,
    subject,
    body_text
) VALUES (
    1,                                    -- Your mail_settings ID from above
    'sender@example.com',                 -- From address
    'recipient1@example.com, recipient2@example.com',  -- To (comma-separated)
    'cc@example.com',                     -- CC (optional)
    'bcc@example.com',                    -- BCC (optional)
    'Test Email from PostgreSQL',         -- Subject
    'This email was sent asynchronously from PostgreSQL via pgbridge!'  -- Body
) RETURNING id;

-- Trigger the send by notifying pgbridge
NOTIFY pgb_mail, '123';  -- Replace 123 with the actual mail ID returned above
```

**3. Automated Email from Trigger:**

```sql
-- Example: Send email when a critical error occurs
CREATE OR REPLACE FUNCTION notify_admin_on_error()
RETURNS TRIGGER AS $$
DECLARE
    mail_id INTEGER;
    error_details TEXT;
BEGIN
    -- Build error details
    error_details := format(
        'Error occurred at: %s\n\nDetails:\n%s\n\nUser: %s\nTable: %s',
        NOW(),
        NEW.error_message,
        current_user,
        TG_TABLE_NAME
    );

    -- Insert mail record
    INSERT INTO pgb.pgb_mail (
        mail_setting_id,
        header_from,
        header_to,
        subject,
        body_text
    ) VALUES (
        1,  -- Your SMTP settings ID
        'alerts@myapp.com',
        'admin@myapp.com',
        'ALERT: Critical Error in ' || TG_TABLE_NAME,
        error_details
    ) RETURNING id INTO mail_id;

    -- Notify pgbridge to send the email
    PERFORM pg_notify('pgb_mail', mail_id::text);

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Attach trigger to your error log table
CREATE TRIGGER error_notification_trigger
    AFTER INSERT ON error_log
    FOR EACH ROW
    WHEN (NEW.severity = 'CRITICAL')
    EXECUTE FUNCTION notify_admin_on_error();
```

**4. Send Daily Summary Report:**

```sql
-- Example function to send daily summary
CREATE OR REPLACE FUNCTION send_daily_summary()
RETURNS void AS $$
DECLARE
    mail_id INTEGER;
    summary_text TEXT;
    total_orders INTEGER;
    total_revenue NUMERIC;
BEGIN
    -- Gather statistics
    SELECT COUNT(*), SUM(total_amount)
    INTO total_orders, total_revenue
    FROM orders
    WHERE DATE(created_at) = CURRENT_DATE;

    -- Build summary text
    summary_text := format(
        'Daily Summary for %s\n\n' ||
        '================================\n\n' ||
        'Total Orders: %s\n' ||
        'Total Revenue: $%s\n\n' ||
        '================================\n\n' ||
        'This is an automated report from pgbridge.',
        CURRENT_DATE,
        total_orders,
        total_revenue
    );

    -- Insert mail and send
    INSERT INTO pgb.pgb_mail (
        mail_setting_id,
        header_from,
        header_to,
        subject,
        body_text
    ) VALUES (
        1,
        'reports@myapp.com',
        'management@myapp.com',
        'Daily Summary Report - ' || CURRENT_DATE,
        summary_text
    ) RETURNING id INTO mail_id;

    PERFORM pg_notify('pgb_mail', mail_id::text);
END;
$$ LANGUAGE plpgsql;

-- Schedule with pg_cron or external scheduler
-- With pg_cron:
-- SELECT cron.schedule('daily-summary', '0 18 * * *', 'SELECT send_daily_summary()');
```

**5. Check Mail Status:**

```sql
-- View all pending emails
SELECT id, header_to, subject, created_at, retry_count, error_message
FROM pgb.pgb_mail
WHERE is_sent = false
ORDER BY created_at DESC;

-- View sent emails
SELECT id, header_to, subject, sent_ts
FROM pgb.pgb_mail
WHERE is_sent = true
ORDER BY sent_ts DESC
LIMIT 20;

-- View failed emails (exceeded retry limit)
SELECT id, header_to, subject, retry_count, error_message, created_at
FROM pgb.pgb_mail
WHERE is_sent = false AND retry_count >= 3
ORDER BY created_at DESC;
```

**6. Resend Failed Emails:**

```sql
-- Retry a specific failed email
-- First, reset the retry count
UPDATE pgb.pgb_mail
SET retry_count = 0, error_message = NULL
WHERE id = 123;

-- Then notify pgbridge to retry
NOTIFY pgb_mail, '123';

-- Or reset ALL failed emails for retry
UPDATE pgb.pgb_mail
SET retry_count = 0, error_message = NULL
WHERE is_sent = false AND retry_count >= 3;

-- pgbridge will automatically process these on next startup via ProcessQueue
```

**7. Multiple SMTP Configurations:**

```sql
-- You can have different SMTP settings for different purposes
-- Transactional emails (fast, reliable)
INSERT INTO pgb.pgb_mail_settings (smtp_server, smtp_port, is_tls, smtp_user, smtp_password)
VALUES ('smtp.sendgrid.net', 587, true, 'apikey', 'your-sendgrid-api-key')
RETURNING id;  -- e.g., returns 1

-- Marketing emails (bulk sender)
INSERT INTO pgb.pgb_mail_settings (smtp_server, smtp_port, is_tls, smtp_user, smtp_password)
VALUES ('smtp.mailgun.org', 587, true, 'postmaster@mg.example.com', 'mailgun-password')
RETURNING id;  -- e.g., returns 2

-- Alert emails (internal SMTP)
INSERT INTO pgb.pgb_mail_settings (smtp_server, smtp_port, is_tls, smtp_user, smtp_password)
VALUES ('mail.company.local', 587, true, 'alerts', 'password')
RETURNING id;  -- e.g., returns 3

-- Use different settings for different email types
INSERT INTO pgb.pgb_mail (mail_setting_id, header_from, header_to, subject, body_text)
VALUES (1, 'noreply@app.com', 'user@example.com', 'Order Confirmation', '...');  -- Uses SendGrid

INSERT INTO pgb.pgb_mail (mail_setting_id, header_from, header_to, subject, body_text)
VALUES (3, 'alerts@company.com', 'ops@company.com', 'Server Alert', '...');  -- Uses internal SMTP
```

**Common SMTP Providers Configuration:**

| Provider | SMTP Server | Port | TLS | SSL | Notes |
|----------|-------------|------|-----|-----|-------|
| Gmail | smtp.gmail.com | 587 | ✓ | ✗ | Requires app-specific password |
| Gmail (SSL) | smtp.gmail.com | 465 | ✗ | ✓ | Alternative SSL config |
| SendGrid | smtp.sendgrid.net | 587 | ✓ | ✗ | Use "apikey" as username |
| Mailgun | smtp.mailgun.org | 587 | ✓ | ✗ | Use postmaster@ address |
| Amazon SES | email-smtp.us-east-1.amazonaws.com | 587 | ✓ | ✗ | Region-specific endpoints |
| Office 365 | smtp.office365.com | 587 | ✓ | ✗ | Modern auth required |
| Outlook.com | smtp-mail.outlook.com | 587 | ✓ | ✗ | Personal accounts |

**Troubleshooting:**

```sql
-- Check recent errors
SELECT id, header_to, subject, error_message, retry_count, created_at
FROM pgb.pgb_mail
WHERE error_message IS NOT NULL
ORDER BY created_at DESC
LIMIT 10;

-- Test SMTP settings (insert a test email)
INSERT INTO pgb.pgb_mail (mail_setting_id, header_from, header_to, subject, body_text)
VALUES (1, 'test@example.com', 'your-email@example.com', 'Test Email', 'This is a test.')
RETURNING id;

-- Watch pgbridge logs for errors
-- In another terminal: journalctl -u pgbridge -f

-- Common errors:
-- 1. "Authentication failed" - Check username/password in pgb_mail_settings
-- 2. "Connection timeout" - Check smtp_server and smtp_port, verify firewall rules
-- 3. "TLS handshake failed" - Try switching between is_tls and is_ssl settings
```

### Using pgb_notify Module

The `pgb_notify` module forwards notifications from individual databases to a central notification hub (`pansoinco_suite`). This enables a unified notification system across multiple databases.

**Architecture:**
- Source databases write to `pgb.pgb_notify`
- pgbridge forwards to `public.ps_notifications` in central database
- One-way sync: notifications are sent to central and users view/manage them there
- Source database tracks whether notification was successfully sent (is_sent, sent_ts)

**1. Setup Central Database:**

First, create the central notifications table in `pansoinco_suite`:

```sql
-- Connect to pansoinco_suite database
\c pansoinco_suite

CREATE TABLE IF NOT EXISTS public.ps_notifications (
    id SERIAL PRIMARY KEY,
    user_email VARCHAR NOT NULL,
    received_ts TIMESTAMP,
    sender_db VARCHAR NOT NULL,
    original_id INT NOT NULL,
    message TEXT,
    message_link VARCHAR,
    is_seen BOOLEAN DEFAULT false NOT NULL,
    seen_ts TIMESTAMP,
    criticality SMALLINT DEFAULT 1 NOT NULL,
    CONSTRAINT ps_notifications_pk PRIMARY KEY (id)
);

CREATE INDEX idx_ps_notifications_user_email ON public.ps_notifications(user_email);
CREATE INDEX idx_ps_notifications_is_seen ON public.ps_notifications(is_seen) WHERE is_seen = false;
CREATE INDEX idx_ps_notifications_sender_original ON public.ps_notifications(sender_db, original_id);

COMMENT ON TABLE public.ps_notifications IS 'Centralized notifications from all databases';
```

**2. Configure Central Database Connection:**

Create `/etc/pgbridge/central.conf`:

```
# Central notification database connection
postgres://pgb:password@central-host:5432/pansoinco_suite?sslmode=require
```

**3. Send a Notification from Source Database:**

```sql
-- In your application database (e.g., "River")
-- Simply insert a notification - the trigger automatically handles the rest!
INSERT INTO pgb.pgb_notify (
    user_email,
    sender_db,
    message,
    message_link,
    criticality
) VALUES (
    'user@example.com',           -- Recipient email
    'River',                       -- Source database name
    'New order #12345 requires approval',  -- Notification message
    'http://app.example.com/orders/12345', -- Link to related resource
    3                              -- Criticality: 1=Info, 2=Low, 3=Medium, 4=High, 5=Critical
);

-- That's it! The S01_send_notification trigger automatically:
-- 1. Sends NOTIFY pgb_notify with the notification ID
-- 2. pgbridge receives the NOTIFY and forwards to central database
-- 3. pgbridge marks is_sent=true and sets sent_ts after successful forwarding
```

**4. Automated Notifications from Trigger:**

```sql
-- Example: Notify users when their order status changes
CREATE OR REPLACE FUNCTION notify_order_status_change()
RETURNS TRIGGER AS $$
DECLARE
    notification_message TEXT;
BEGIN
    -- Build notification message
    notification_message := format(
        'Order #%s status changed from %s to %s',
        NEW.order_number,
        OLD.status,
        NEW.status
    );

    -- Just insert the notification - the pgb trigger handles the rest!
    INSERT INTO pgb.pgb_notify (
        user_email,
        sender_db,
        message,
        message_link,
        criticality
    ) VALUES (
        NEW.customer_email,
        'River',  -- Current database name
        notification_message,
        'http://app.example.com/orders/' || NEW.order_number,
        CASE
            WHEN NEW.status = 'shipped' THEN 2
            WHEN NEW.status = 'cancelled' THEN 4
            ELSE 1
        END
    );
    -- No need to call NOTIFY manually - S01_send_notification trigger does it!

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER order_status_notification
    AFTER UPDATE OF status ON orders
    FOR EACH ROW
    WHEN (OLD.status IS DISTINCT FROM NEW.status)
    EXECUTE FUNCTION notify_order_status_change();
```

**5. Check Notification Status:**

```sql
-- In source database: View pending/sent notifications
SELECT id, user_email, message, created_at, is_sent, sent_ts
FROM pgb.pgb_notify
WHERE is_sent = false  -- Only show pending notifications
ORDER BY created_at DESC;

-- In central database: View all notifications
SELECT id, user_email, sender_db, message, received_ts, is_seen
FROM public.ps_notifications
WHERE user_email = 'user@example.com'
ORDER BY received_ts DESC
LIMIT 20;

-- In central database: View unread notifications
SELECT id, user_email, sender_db, message, received_ts, criticality
FROM public.ps_notifications
WHERE user_email = 'user@example.com'
  AND is_seen = false
ORDER BY criticality DESC, received_ts DESC;
```

**6. Mark Notification as Seen (in Central Database):**

```sql
-- When user views a notification in the central system
UPDATE public.ps_notifications
SET is_seen = true,
    seen_ts = CURRENT_TIMESTAMP
WHERE id = 456;  -- Central notification ID

-- Note: Seen status is only tracked in central database
-- Source database only tracks if notification was sent (is_sent, sent_ts)
```

**7. Bulk Operations:**

```sql
-- Send notifications to multiple users
DO $$
DECLARE
    user_record RECORD;
    notify_id INTEGER;
BEGIN
    FOR user_record IN
        SELECT email FROM users WHERE department = 'Sales'
    LOOP
        INSERT INTO pgb.pgb_notify (
            user_email,
            sender_db,
            message,
            criticality
        ) VALUES (
            user_record.email,
            'River',
            'Monthly sales report is now available',
            1
        ) RETURNING id INTO notify_id;

        PERFORM pg_notify('pgb_notify', notify_id::text);
    END LOOP;
END $$;
```

**8. Notification Criticality Levels:**

| Level | Name | Use Case | Example |
|-------|------|----------|---------|
| 1 | Info | General information | "Report generated" |
| 2 | Low | Minor updates | "Comment added" |
| 3 | Medium | Requires attention | "Order pending approval" |
| 4 | High | Urgent action needed | "Payment failed" |
| 5 | Critical | Immediate action required | "Security alert" |

**9. Query Notifications by Criticality:**

```sql
-- Get high-priority unread notifications across all databases
SELECT sender_db, user_email, message, message_link, received_ts
FROM public.ps_notifications
WHERE is_seen = false
  AND criticality >= 4
ORDER BY criticality DESC, received_ts DESC;

-- Count notifications by database and criticality
SELECT sender_db, criticality, COUNT(*) as notification_count
FROM public.ps_notifications
WHERE is_seen = false
GROUP BY sender_db, criticality
ORDER BY sender_db, criticality;
```

**10. Monitoring and Maintenance:**

```sql
-- Check forwarding status in source database
SELECT
    COUNT(*) FILTER (WHERE is_sent = false) as pending_forward,
    COUNT(*) FILTER (WHERE is_sent = true) as forwarded
FROM pgb.pgb_notify;

-- Cleanup old sent notifications (older than 90 days)
DELETE FROM pgb.pgb_notify
WHERE is_sent = true
  AND sent_ts < CURRENT_TIMESTAMP - INTERVAL '90 days';

-- In central database: Archive old notifications
INSERT INTO public.ps_notifications_archive
SELECT * FROM public.ps_notifications
WHERE is_seen = true
  AND seen_ts < CURRENT_TIMESTAMP - INTERVAL '180 days';

DELETE FROM public.ps_notifications
WHERE is_seen = true
  AND seen_ts < CURRENT_TIMESTAMP - INTERVAL '180 days';
```

**Troubleshooting:**

```sql
-- Check if notifications are being forwarded
SELECT id, user_email, message, created_at, is_sent
FROM pgb.pgb_notify
WHERE is_sent = false
ORDER BY created_at DESC
LIMIT 10;

-- Verify connection to central database
-- Check pgbridge logs: journalctl -u pgbridge -f

-- Manually retry failed forwards
UPDATE pgb.pgb_notify
SET is_sent = false
WHERE id IN (SELECT id FROM pgb.pgb_notify WHERE is_sent = false);

-- Then notify pgbridge for each:
-- NOTIFY pgb_notify, '<id>';

-- Common issues:
-- 1. Notifications not appearing in central DB
--    - Check central database connection in /etc/pgbridge/central.conf
--    - Verify pgb user has INSERT permission on ps_notifications
--    - Check pgbridge logs for connection errors

-- 2. "Seen" status not syncing back
--    - Verify pgbridge is running and connected to both databases
--    - Check polling is working (every 10 seconds by default)
--    - Ensure sender_db matches database name in config file

-- 3. Duplicate notifications in central DB
--    - Check for multiple pgbridge instances running
--    - Verify idempotency in your trigger logic
```

**Best Practices:**

1. **Use meaningful sender_db names**: Match the database name in pgbridge config exactly
2. **Include message_link**: Always provide a link to the relevant resource
3. **Set appropriate criticality**: Use criticality levels consistently across all databases
4. **Clean up old notifications**: Implement regular cleanup to prevent table bloat
5. **Monitor forwarding lag**: Alert if unsent notifications accumulate
6. **Test bidirectional sync**: Verify "seen" status syncs back to source databases

## Configuration

pgbridge supports two configuration methods:

1. **Database-based Configuration** (Recommended) - Centralized configuration in `pansoinco_suite`
2. **File-based Configuration** (Legacy) - Traditional configuration file

### Database-Based Configuration (Recommended)

Load database connections dynamically from the central `pansoinco_suite` database:

**Benefits:**
- ✅ Centralized management - all configurations in one place
- ✅ Dynamic updates - no need to restart pgbridge for config changes
- ✅ Integrated with existing PanSoinco infrastructure
- ✅ Audit trail - all config changes tracked in database

**Setup:**

1. Configure your databases in `pansoinco_suite`:
   ```sql
   -- Databases are managed via sw_instance, sw_pgb, and ps_sw tables
   -- The view automatically formats them for pgbridge

   -- View the current configuration:
   SELECT
       ps.short_name || ' ' || si.db_name as "Database Name",
       si.db_connection_string as "Connection String",
       '[' || array_to_string(sp.pgb_services, ',') || ']' as "PGB Services"
   FROM sw_pgb sp
   LEFT JOIN sw_instance si ON si.id = sp.sw_instance_id
   LEFT JOIN ps_sw ps ON ps.id = si.sw_id
   WHERE sp.pgb_services IS NOT NULL;
   ```

2. Run pgbridge with database config:
   ```bash
   # Use default central config (/etc/pgbridge/central.conf)
   pgbridge --db-config

   # Or specify custom central config path
   pgbridge --db-config /path/to/central.conf
   ```

3. Central config file (`/etc/pgbridge/central.conf`):
   ```
   # Connection string to pansoinco_suite
   postgres://pgb:password@central-host:5432/pansoinco_suite?sslmode=require
   ```

### File-Based Configuration (Legacy)

Create `/etc/pgbridge/pgbridge.conf`:

```
# Format: database_name, connection_string, [module1, module2, ...]
# Lines starting with # are comments

# Example with mail and notify modules
production_db, postgres://pgb:password@db.example.com:5432/production?sslmode=require, [pgb_mail, pgb_notify]

# Example with async module (use with caution)
analytics_db, postgres://pgb:password@localhost:5432/analytics, [pgb_async, pgb_csv]

# Example with minimal modules
app_db, postgres://pgb:password@10.0.1.50:5432/myapp, [pgb_mail]
```

Run with file config:
```bash
pgbridge /etc/pgbridge/pgbridge.conf
```

**Configuration Notes:**
- Database names must be unique
- Connection strings support standard PostgreSQL connection URIs
- At least one module must be specified per database
- Complex connection strings with query parameters are supported

### Security Best Practices

1. **Use SSL/TLS for connections:**
   ```
   postgres://pgb:pass@host:5432/db?sslmode=require
   ```

2. **Restrict configuration file permissions:**
   ```bash
   chmod 600 /etc/pgbridge/pgbridge.conf
   chown pgbridge:pgbridge /etc/pgbridge/pgbridge.conf
   ```

3. **Use strong passwords:**
   - Minimum 16 characters
   - Mix of letters, numbers, and symbols
   - Consider using a password manager or secrets management system

4. **Network security:**
   - Configure `pg_hba.conf` to restrict connections
   - Use host-based authentication
   - Consider using certificate-based authentication

5. **For pgb_async module:**
   - Audit all SQL statements before execution
   - Implement application-level access controls
   - Log all async operations
   - Consider using database audit extensions

## Installation

### Building from Source

```bash
# Clone the repository
git clone https://github.com/yourusername/pgbridge.git
cd pgbridge

# Build the binary
go build -o bin/pgbridge cmd/pgbridge/main.go

# Install (requires root)
sudo cp bin/pgbridge /usr/local/bin/
sudo chmod +x /usr/local/bin/pgbridge

# Create configuration directory
sudo mkdir -p /etc/pgbridge
sudo cp config/pgbridge.conf.example /etc/pgbridge/pgbridge.conf
sudo chmod 600 /etc/pgbridge/pgbridge.conf
```

### Systemd Service (Linux)

Create `/etc/systemd/system/pgbridge.service`:

```ini
[Unit]
Description=PostgreSQL Bridge Service
After=network.target postgresql.service
Wants=postgresql.service

[Service]
Type=simple
User=pgbridge
Group=pgbridge
ExecStart=/usr/local/bin/pgbridge -config /etc/pgbridge/pgbridge.conf
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/pgbridge

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl daemon-reload
sudo systemctl enable pgbridge
sudo systemctl start pgbridge
sudo systemctl status pgbridge
```

## Running Tests

### Unit Tests (Config Parser)

No database required:
```bash
go test ./internal/config/... -v
```

### Integration Tests (Database Connection)

Requires PostgreSQL:
```bash
# Create test database
psql -U postgres -c "CREATE DATABASE pgbridge_test;"

# Run tests
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/pgbridge_test?sslmode=disable" \
    go test ./internal/database/... -v
```

### All Tests

```bash
# Unit tests
go test ./internal/config/... -v

# Integration tests
TEST_DATABASE_URL="postgres://user:pass@localhost:5432/pgbridge_test?sslmode=disable" \
    go test ./internal/database/... -v
```

## Quick Start Checklist

Before adding a database to pgbridge:

- [ ] Create `pgb` role with strong password
- [ ] Grant `CONNECT` on target database
- [ ] Grant `CREATE` on target database (for pgb schema)
- [ ] Review which modules you need
- [ ] Grant module-specific permissions (see sections above)
- [ ] If using `pgb_async`: **Carefully review and restrict permissions**
- [ ] Test connection manually: `psql -U pgb -h hostname -d database`
- [ ] Add entry to `/etc/pgbridge/pgbridge.conf`
- [ ] Restart pgbridge service
- [ ] Check logs: `journalctl -u pgbridge -f`
- [ ] Verify `pgb` schema and `pgb_log` table were created
- [ ] Check for `SERVICE_START` entry in `pgb.pgb_log`

## Monitoring and Troubleshooting

### Check Service Status
```bash
systemctl status pgbridge
journalctl -u pgbridge -f
```

### Verify Database Connection
```sql
-- Connect to your database
\c your_database_name

-- Check if pgb schema exists
\dn pgb

-- Check pgb_log for service events
SELECT * FROM pgb.pgb_log
WHERE event_type = 'SERVICE_START'
ORDER BY timestamp DESC
LIMIT 10;

-- Monitor recent activity
SELECT event_type, database_name, module_name, message, timestamp
FROM pgb.pgb_log
ORDER BY timestamp DESC
LIMIT 50;
```

### Common Issues

1. **Connection refused:**
   - Check `pg_hba.conf` for host-based authentication
   - Verify firewall rules
   - Confirm PostgreSQL is listening on correct interface

2. **Permission denied on schema creation:**
   - Grant `CREATE` on database to pgb role
   - Check if schema already exists with different owner

3. **Permission denied on tables:**
   - Review module-specific permissions
   - Check `GRANT` statements were executed correctly
   - Verify schema ownership

4. **pgb_async SQL execution fails:**
   - Check pgb role has permissions on target tables
   - Review SQL in `pgb.pgb_async` table for syntax errors
   - Check `pgb.pgb_log` for detailed error messages

## License

[Your License Here]

## Contributing

[Contributing Guidelines]

## Support

For issues and questions:
- GitHub Issues: [your-repo-url]
- Documentation: [docs-url]
>>>>>>> 370c7e6 (Initial commit)
