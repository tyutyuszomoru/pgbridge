-- Fix: connection_string(db_owner, db_password, db_host, db_port, db_name) was wrapping
-- the password in single quotes, producing postgres://user:'pass'@host which breaks URI parsing.

CREATE OR REPLACE FUNCTION public.connection_string(
    db_owner   character varying,
    db_password character varying,
    db_host    character varying,
    db_port    integer,
    db_name    character varying
) RETURNS character varying
    LANGUAGE plpgsql AS
$$
DECLARE
    v_ret varchar;
BEGIN
    v_ret := 'postgres://' || db_owner || ':' || db_password || '@'
             || db_host || ':' || db_port || '/' || db_name;
    RETURN v_ret;
END;
$$;

-- Re-trigger connection_string rebuild on all existing sw_instance rows
UPDATE public.sw_instance SET db_host = db_host;

-- Re-trigger connection_string rebuild on all existing sw_pgb rows
UPDATE public.sw_pgb SET sw_instance_id = sw_instance_id;
