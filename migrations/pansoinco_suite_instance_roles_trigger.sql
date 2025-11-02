-- Trigger for automatic role discovery on new instance creation
-- This trigger sends a notification when a new instance is created
-- pgbridge will listen to this notification and discover/populate roles

-- Create the trigger function (in public schema)
CREATE OR REPLACE FUNCTION trg_instance_roles_notify()
RETURNS TRIGGER AS $$
BEGIN
    -- Send NOTIFY with the new instance ID
    PERFORM pg_notify('pgb_instance_roles', NEW.id::text);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Drop existing trigger if it exists
DROP TRIGGER IF EXISTS S01_instance_roles_notify ON sw_instance;

-- Create the trigger on sw_instance
CREATE TRIGGER S01_instance_roles_notify
AFTER INSERT ON sw_instance
FOR EACH ROW
EXECUTE FUNCTION trg_instance_roles_notify();

-- Verify the trigger was created
SELECT
    tgname as trigger_name,
    relname as table_name,
    proname as function_name,
    tgenabled as enabled
FROM pg_trigger t
JOIN pg_class c ON t.tgrelid = c.oid
JOIN pg_proc p ON t.tgfoid = p.oid
WHERE tgname = 'S01_instance_roles_notify';
