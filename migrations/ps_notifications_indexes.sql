-- Indexes for public.ps_notifications in pansoinco_suite
-- Run once on the central database after creating the table.

CREATE INDEX IF NOT EXISTS idx_ps_notifications_user_email
    ON public.ps_notifications(user_email);

CREATE INDEX IF NOT EXISTS idx_ps_notifications_is_seen
    ON public.ps_notifications(is_seen) WHERE is_seen = false;

-- Allows efficient lookup when checking for duplicates or tracing back to source
CREATE INDEX IF NOT EXISTS idx_ps_notifications_sender_original
    ON public.ps_notifications(sender_db, original_id);
