--
-- PostgreSQL database cluster dump
--

-- Started on 2025-11-02 11:19:29 EST

SET default_transaction_read_only = off;

SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;

--
-- Roles
--

CREATE ROLE boat_owners;
ALTER ROLE boat_owners WITH NOSUPERUSER NOINHERIT NOCREATEROLE NOCREATEDB NOLOGIN NOREPLICATION NOBYPASSRLS;
CREATE ROLE postgres;
ALTER ROLE postgres WITH SUPERUSER INHERIT CREATEROLE CREATEDB LOGIN REPLICATION BYPASSRLS;
CREATE ROLE riverside;
ALTER ROLE riverside WITH NOSUPERUSER INHERIT NOCREATEROLE NOCREATEDB LOGIN NOREPLICATION NOBYPASSRLS;
CREATE ROLE tyutyu;
ALTER ROLE tyutyu WITH NOSUPERUSER INHERIT CREATEROLE CREATEDB LOGIN NOREPLICATION NOBYPASSRLS;

--
-- User Configurations
--


--
-- Role memberships
--

GRANT boat_owners TO tyutyu WITH ADMIN OPTION, INHERIT FALSE, SET FALSE GRANTED BY postgres;
GRANT riverside TO tyutyu WITH ADMIN OPTION, INHERIT FALSE, SET FALSE GRANTED BY postgres;






--
-- Databases
--

--
-- Database "template1" dump
--

\connect template1

--
-- PostgreSQL database dump
--

-- Dumped from database version 17.5
-- Dumped by pg_dump version 17.5

-- Started on 2025-11-02 11:19:29 EST

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

-- Completed on 2025-11-02 11:19:30 EST

--
-- PostgreSQL database dump complete
--

--
-- Database "pansoinco_suite" dump
--

--
-- PostgreSQL database dump
--

-- Dumped from database version 17.5
-- Dumped by pg_dump version 17.5

-- Started on 2025-11-02 11:19:30 EST

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- TOC entry 4374 (class 1262 OID 18261)
-- Name: pansoinco_suite; Type: DATABASE; Schema: -; Owner: tyutyu
--

CREATE DATABASE pansoinco_suite WITH TEMPLATE = template0 ENCODING = 'UTF8' LOCALE_PROVIDER = libc LOCALE = 'en_US.UTF-8';


ALTER DATABASE pansoinco_suite OWNER TO tyutyu;

\connect pansoinco_suite

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- TOC entry 6 (class 2615 OID 19365)
-- Name: pgb; Type: SCHEMA; Schema: -; Owner: tyutyu
--

CREATE SCHEMA pgb;


ALTER SCHEMA pgb OWNER TO tyutyu;

--
-- TOC entry 901 (class 1247 OID 19324)
-- Name: email; Type: DOMAIN; Schema: public; Owner: tyutyu
--

CREATE DOMAIN public.email AS text
	CONSTRAINT email_check CHECK ((VALUE ~ '^\w+@[a-zA-Z_]+?\.[a-zA-Z]{2,3}$'::text));


ALTER DOMAIN public.email OWNER TO tyutyu;

--
-- TOC entry 886 (class 1247 OID 19244)
-- Name: pgb_services; Type: TYPE; Schema: public; Owner: tyutyu
--

CREATE TYPE public.pgb_services AS ENUM (
    'pgb_mail',
    'pgb_notify',
    'pgb_async',
    'pgb_csv',
    'pgb_file',
    'pgb_excel',
    'pgb_instance_roles'
);


ALTER TYPE public.pgb_services OWNER TO tyutyu;

--
-- TOC entry 877 (class 1247 OID 19203)
-- Name: swstatus; Type: TYPE; Schema: public; Owner: tyutyu
--

CREATE TYPE public.swstatus AS ENUM (
    'development',
    'quality',
    'production'
);


ALTER TYPE public.swstatus OWNER TO tyutyu;

--
-- TOC entry 238 (class 1255 OID 19287)
-- Name: connection_string(character varying, character varying, integer, character varying, character varying); Type: FUNCTION; Schema: public; Owner: tyutyu
--

CREATE FUNCTION public.connection_string(db_name character varying, db_host character varying, db_port integer, db_owner character varying, db_password character varying) RETURNS character varying
    LANGUAGE plpgsql
    AS $$
	DECLARE 
		v_ret varchar;
	BEGIN
	
	v_ret := 'postgres://' || db_owner || ':' || db_password || '@' || db_host || ':' || db_port || '/' || db_name;

	RETURN v_ret;

	END;
$$;


ALTER FUNCTION public.connection_string(db_name character varying, db_host character varying, db_port integer, db_owner character varying, db_password character varying) OWNER TO tyutyu;

--
-- TOC entry 239 (class 1255 OID 19288)
-- Name: connection_string(character varying, character varying, character varying, integer, character varying); Type: FUNCTION; Schema: public; Owner: tyutyu
--

CREATE FUNCTION public.connection_string(db_owner character varying, db_password character varying, db_host character varying, db_port integer, db_name character varying) RETURNS character varying
    LANGUAGE plpgsql
    AS $$
	DECLARE 
		v_ret varchar;
	BEGIN
	
	v_ret := 'postgres://' || db_owner || ':''' || db_password || '''@' || db_host || ':' || db_port || '/' || db_name;

	RETURN v_ret;

	END;
$$;


ALTER FUNCTION public.connection_string(db_owner character varying, db_password character varying, db_host character varying, db_port integer, db_name character varying) OWNER TO tyutyu;

--
-- TOC entry 4375 (class 0 OID 0)
-- Dependencies: 239
-- Name: FUNCTION connection_string(db_owner character varying, db_password character varying, db_host character varying, db_port integer, db_name character varying); Type: COMMENT; Schema: public; Owner: tyutyu
--

COMMENT ON FUNCTION public.connection_string(db_owner character varying, db_password character varying, db_host character varying, db_port integer, db_name character varying) IS 'db_owner, db_password, db_host, db_port , db_name';


--
-- TOC entry 242 (class 1255 OID 19362)
-- Name: trg_instance_roles_notify(); Type: FUNCTION; Schema: public; Owner: tyutyu
--

CREATE FUNCTION public.trg_instance_roles_notify() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    -- Send NOTIFY with the new instance ID
    PERFORM pg_notify('pgb_instance_roles', NEW.id::text);
    RETURN NEW;
END;
$$;


ALTER FUNCTION public.trg_instance_roles_notify() OWNER TO tyutyu;

--
-- TOC entry 240 (class 1255 OID 19285)
-- Name: trg_pgb_connection_string(); Type: FUNCTION; Schema: public; Owner: tyutyu
--

CREATE FUNCTION public.trg_pgb_connection_string() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
	DECLARE
		v_host varchar;
		v_port integer;
		v_database varchar;
	BEGIN
/*
ZRB 2025-11-01 if no username and password is given to pgb, it uses connection string from instance. Otherwise creates new from instance data and own user/pass
*/

	IF NEW.pgb_role IS NULL OR NEW.pgb_password IS NULL THEN
		SELECT
			db_owner,
			db_password,
			db_connection_string
		INTO
			NEW.pgb_role,
			NEW.pgb_password,
			NEW.connection_string
		FROM
			public.sw_instance
		WHERE
			id = NEW.sw_instance_id;

		RETURN NEW;
	END IF;

	SELECT
		public.connection_string(
			NEW.pgb_role, 
			NEW.pgb_password ,
			s.db_host,
			s.db_port,
			s.db_name
		)
	INTO
		NEW.connection_string
	FROM
		sw_instance s
	WHERE
		s.id = NEW.sw_instance_id;

	RETURN NEW;

	END;
$$;


ALTER FUNCTION public.trg_pgb_connection_string() OWNER TO tyutyu;

--
-- TOC entry 241 (class 1255 OID 19289)
-- Name: trg_sw_instance_connection_string(); Type: FUNCTION; Schema: public; Owner: tyutyu
--

CREATE FUNCTION public.trg_sw_instance_connection_string() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
	BEGIN
/*
ZRB 2025-11-01 Created
*/

	NEW.db_connection_string = public.connection_string(NEW.db_owner, NEW.db_password, NEW.db_host, NEW.db_port, NEW.db_name);
	
	RETURN NEW;

	END;
$$;


ALTER FUNCTION public.trg_sw_instance_connection_string() OWNER TO tyutyu;

--
-- TOC entry 4376 (class 0 OID 0)
-- Dependencies: 241
-- Name: FUNCTION trg_sw_instance_connection_string(); Type: COMMENT; Schema: public; Owner: tyutyu
--

COMMENT ON FUNCTION public.trg_sw_instance_connection_string() IS 'connection_string field creation';


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- TOC entry 237 (class 1259 OID 19367)
-- Name: pgb_log; Type: TABLE; Schema: pgb; Owner: tyutyu
--

CREATE TABLE pgb.pgb_log (
    id integer NOT NULL,
    "timestamp" timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    service_name character varying(50) DEFAULT 'pgbridge'::character varying,
    event_type character varying(50) NOT NULL,
    database_name character varying(100),
    module_name character varying(50),
    message text,
    details jsonb
);


ALTER TABLE pgb.pgb_log OWNER TO tyutyu;

--
-- TOC entry 236 (class 1259 OID 19366)
-- Name: pgb_log_id_seq; Type: SEQUENCE; Schema: pgb; Owner: tyutyu
--

CREATE SEQUENCE pgb.pgb_log_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE pgb.pgb_log_id_seq OWNER TO tyutyu;

--
-- TOC entry 4377 (class 0 OID 0)
-- Dependencies: 236
-- Name: pgb_log_id_seq; Type: SEQUENCE OWNED BY; Schema: pgb; Owner: tyutyu
--

ALTER SEQUENCE pgb.pgb_log_id_seq OWNED BY pgb.pgb_log.id;


--
-- TOC entry 219 (class 1259 OID 18263)
-- Name: ps_notifications; Type: TABLE; Schema: public; Owner: tyutyu
--

CREATE TABLE public.ps_notifications (
    id integer NOT NULL,
    user_email character varying NOT NULL,
    received_ts timestamp without time zone,
    sender_db character varying,
    message text,
    message_link character varying,
    is_seen boolean DEFAULT false NOT NULL,
    seen_ts timestamp without time zone,
    criticality smallint DEFAULT 1 NOT NULL,
    original_id integer NOT NULL
);


ALTER TABLE public.ps_notifications OWNER TO tyutyu;

--
-- TOC entry 4378 (class 0 OID 0)
-- Dependencies: 219
-- Name: TABLE ps_notifications; Type: COMMENT; Schema: public; Owner: tyutyu
--

COMMENT ON TABLE public.ps_notifications IS 'Notifications sent by other databases';


--
-- TOC entry 4379 (class 0 OID 0)
-- Dependencies: 219
-- Name: COLUMN ps_notifications.criticality; Type: COMMENT; Schema: public; Owner: tyutyu
--

COMMENT ON COLUMN public.ps_notifications.criticality IS 'criticality level to set up (1-5 increasing)';


--
-- TOC entry 218 (class 1259 OID 18262)
-- Name: notify_id_seq; Type: SEQUENCE; Schema: public; Owner: tyutyu
--

CREATE SEQUENCE public.notify_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.notify_id_seq OWNER TO tyutyu;

--
-- TOC entry 4380 (class 0 OID 0)
-- Dependencies: 218
-- Name: notify_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: tyutyu
--

ALTER SEQUENCE public.notify_id_seq OWNED BY public.ps_notifications.id;


--
-- TOC entry 227 (class 1259 OID 19291)
-- Name: ps_auth_token; Type: TABLE; Schema: public; Owner: tyutyu
--

CREATE TABLE public.ps_auth_token (
    email character varying,
    access_token character varying NOT NULL,
    issued timestamp without time zone NOT NULL,
    expires timestamp without time zone NOT NULL,
    is_expired boolean
);


ALTER TABLE public.ps_auth_token OWNER TO tyutyu;

--
-- TOC entry 222 (class 1259 OID 19210)
-- Name: ps_sw; Type: TABLE; Schema: public; Owner: tyutyu
--

CREATE TABLE public.ps_sw (
    id integer NOT NULL,
    short_name character varying(16) NOT NULL,
    long_name character varying
);


ALTER TABLE public.ps_sw OWNER TO tyutyu;

--
-- TOC entry 4381 (class 0 OID 0)
-- Dependencies: 222
-- Name: TABLE ps_sw; Type: COMMENT; Schema: public; Owner: tyutyu
--

COMMENT ON TABLE public.ps_sw IS 'Pansoinco softwares';


--
-- TOC entry 221 (class 1259 OID 19209)
-- Name: ps_sw_id_seq; Type: SEQUENCE; Schema: public; Owner: tyutyu
--

CREATE SEQUENCE public.ps_sw_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.ps_sw_id_seq OWNER TO tyutyu;

--
-- TOC entry 4382 (class 0 OID 0)
-- Dependencies: 221
-- Name: ps_sw_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: tyutyu
--

ALTER SEQUENCE public.ps_sw_id_seq OWNED BY public.ps_sw.id;


--
-- TOC entry 220 (class 1259 OID 19188)
-- Name: ps_users; Type: TABLE; Schema: public; Owner: tyutyu
--

CREATE TABLE public.ps_users (
    email public.email NOT NULL,
    first_name character varying,
    last_name character varying,
    password character varying,
    ps_superuser boolean DEFAULT false NOT NULL,
    created_by character varying,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    last_login timestamp without time zone,
    parent_user character varying,
    totp_enabled boolean DEFAULT true NOT NULL,
    totp_enforced boolean DEFAULT false NOT NULL,
    totp_secret character varying,
    totp_backup character varying[],
    oauth_enabled boolean DEFAULT false NOT NULL,
    oauth_enforced boolean DEFAULT false NOT NULL,
    oauth_provider character varying,
    oauth_userid character varying,
    oauth_access_token character varying,
    oauth_refresh_token character varying,
    oauth_token_expires timestamp with time zone,
    is_user_disabled boolean DEFAULT false NOT NULL
);


ALTER TABLE public.ps_users OWNER TO tyutyu;

--
-- TOC entry 4383 (class 0 OID 0)
-- Dependencies: 220
-- Name: TABLE ps_users; Type: COMMENT; Schema: public; Owner: tyutyu
--

COMMENT ON TABLE public.ps_users IS 'All users with access to any Pansoinco SW';


--
-- TOC entry 224 (class 1259 OID 19232)
-- Name: sw_instance; Type: TABLE; Schema: public; Owner: tyutyu
--

CREATE TABLE public.sw_instance (
    id integer NOT NULL,
    sw_id integer NOT NULL,
    instance_name character varying,
    instance_type public.swstatus DEFAULT 'development'::public.swstatus NOT NULL,
    db_name character varying(20),
    db_host character varying DEFAULT 'localhost'::character varying NOT NULL,
    db_port smallint DEFAULT 5432 NOT NULL,
    db_owner character varying NOT NULL,
    db_password character varying NOT NULL,
    db_connection_string character varying,
    rel_docroot character varying NOT NULL,
    uri character varying NOT NULL
);


ALTER TABLE public.sw_instance OWNER TO tyutyu;

--
-- TOC entry 4384 (class 0 OID 0)
-- Dependencies: 224
-- Name: COLUMN sw_instance.rel_docroot; Type: COMMENT; Schema: public; Owner: tyutyu
--

COMMENT ON COLUMN public.sw_instance.rel_docroot IS 'relative path to docroot';


--
-- TOC entry 223 (class 1259 OID 19231)
-- Name: sw_instance_id_seq; Type: SEQUENCE; Schema: public; Owner: tyutyu
--

CREATE SEQUENCE public.sw_instance_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.sw_instance_id_seq OWNER TO tyutyu;

--
-- TOC entry 4385 (class 0 OID 0)
-- Dependencies: 223
-- Name: sw_instance_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: tyutyu
--

ALTER SEQUENCE public.sw_instance_id_seq OWNED BY public.sw_instance.id;


--
-- TOC entry 231 (class 1259 OID 19314)
-- Name: sw_instance_menu; Type: TABLE; Schema: public; Owner: tyutyu
--

CREATE TABLE public.sw_instance_menu (
    id integer NOT NULL,
    sw_instance_id integer NOT NULL,
    menu_title character varying,
    is_uri boolean DEFAULT false NOT NULL,
    superior_menu integer,
    menu_icon character varying,
    filename character varying,
    get_args character varying,
    ordering smallint DEFAULT 1 NOT NULL
);


ALTER TABLE public.sw_instance_menu OWNER TO tyutyu;

--
-- TOC entry 4386 (class 0 OID 0)
-- Dependencies: 231
-- Name: COLUMN sw_instance_menu.filename; Type: COMMENT; Schema: public; Owner: tyutyu
--

COMMENT ON COLUMN public.sw_instance_menu.filename IS 'relative to rel_docroot of instance';


--
-- TOC entry 4387 (class 0 OID 0)
-- Dependencies: 231
-- Name: COLUMN sw_instance_menu.get_args; Type: COMMENT; Schema: public; Owner: tyutyu
--

COMMENT ON COLUMN public.sw_instance_menu.get_args IS 'optional get arguments as key=value';


--
-- TOC entry 230 (class 1259 OID 19313)
-- Name: sw_instance_menu_id_seq; Type: SEQUENCE; Schema: public; Owner: tyutyu
--

CREATE SEQUENCE public.sw_instance_menu_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.sw_instance_menu_id_seq OWNER TO tyutyu;

--
-- TOC entry 4388 (class 0 OID 0)
-- Dependencies: 230
-- Name: sw_instance_menu_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: tyutyu
--

ALTER SEQUENCE public.sw_instance_menu_id_seq OWNED BY public.sw_instance_menu.id;


--
-- TOC entry 235 (class 1259 OID 19346)
-- Name: sw_instance_role_menu; Type: TABLE; Schema: public; Owner: tyutyu
--

CREATE TABLE public.sw_instance_role_menu (
    id integer NOT NULL,
    instancemenu_id integer,
    instancerole_id integer
);


ALTER TABLE public.sw_instance_role_menu OWNER TO tyutyu;

--
-- TOC entry 234 (class 1259 OID 19345)
-- Name: sw_instance_role_menu_id_seq; Type: SEQUENCE; Schema: public; Owner: tyutyu
--

CREATE SEQUENCE public.sw_instance_role_menu_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.sw_instance_role_menu_id_seq OWNER TO tyutyu;

--
-- TOC entry 4389 (class 0 OID 0)
-- Dependencies: 234
-- Name: sw_instance_role_menu_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: tyutyu
--

ALTER SEQUENCE public.sw_instance_role_menu_id_seq OWNED BY public.sw_instance_role_menu.id;


--
-- TOC entry 229 (class 1259 OID 19305)
-- Name: sw_instance_roles; Type: TABLE; Schema: public; Owner: tyutyu
--

CREATE TABLE public.sw_instance_roles (
    id integer NOT NULL,
    instance_id integer NOT NULL,
    role character varying NOT NULL,
    role_description character varying
);


ALTER TABLE public.sw_instance_roles OWNER TO tyutyu;

--
-- TOC entry 228 (class 1259 OID 19304)
-- Name: sw_instance_roles_id_seq; Type: SEQUENCE; Schema: public; Owner: tyutyu
--

CREATE SEQUENCE public.sw_instance_roles_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.sw_instance_roles_id_seq OWNER TO tyutyu;

--
-- TOC entry 4390 (class 0 OID 0)
-- Dependencies: 228
-- Name: sw_instance_roles_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: tyutyu
--

ALTER SEQUENCE public.sw_instance_roles_id_seq OWNED BY public.sw_instance_roles.id;


--
-- TOC entry 233 (class 1259 OID 19335)
-- Name: sw_instance_user; Type: TABLE; Schema: public; Owner: tyutyu
--

CREATE TABLE public.sw_instance_user (
    id integer NOT NULL,
    user_email public.email NOT NULL,
    instance_id integer NOT NULL,
    role_id integer NOT NULL
);


ALTER TABLE public.sw_instance_user OWNER TO tyutyu;

--
-- TOC entry 232 (class 1259 OID 19334)
-- Name: sw_instance_user_id_seq; Type: SEQUENCE; Schema: public; Owner: tyutyu
--

CREATE SEQUENCE public.sw_instance_user_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.sw_instance_user_id_seq OWNER TO tyutyu;

--
-- TOC entry 4391 (class 0 OID 0)
-- Dependencies: 232
-- Name: sw_instance_user_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: tyutyu
--

ALTER SEQUENCE public.sw_instance_user_id_seq OWNED BY public.sw_instance_user.id;


--
-- TOC entry 226 (class 1259 OID 19258)
-- Name: sw_pgb; Type: TABLE; Schema: public; Owner: tyutyu
--

CREATE TABLE public.sw_pgb (
    id integer NOT NULL,
    sw_instance_id integer NOT NULL,
    pgb_role character varying,
    pgb_password character varying,
    pgb_services public.pgb_services[],
    connection_string character varying
);


ALTER TABLE public.sw_pgb OWNER TO tyutyu;

--
-- TOC entry 4392 (class 0 OID 0)
-- Dependencies: 226
-- Name: COLUMN sw_pgb.pgb_role; Type: COMMENT; Schema: public; Owner: tyutyu
--

COMMENT ON COLUMN public.sw_pgb.pgb_role IS 'if null, it will connect as owner';


--
-- TOC entry 4393 (class 0 OID 0)
-- Dependencies: 226
-- Name: COLUMN sw_pgb.pgb_password; Type: COMMENT; Schema: public; Owner: tyutyu
--

COMMENT ON COLUMN public.sw_pgb.pgb_password IS 'if null, it will connect as owner';


--
-- TOC entry 225 (class 1259 OID 19257)
-- Name: sw_pgb_id_seq; Type: SEQUENCE; Schema: public; Owner: tyutyu
--

CREATE SEQUENCE public.sw_pgb_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.sw_pgb_id_seq OWNER TO tyutyu;

--
-- TOC entry 4394 (class 0 OID 0)
-- Dependencies: 225
-- Name: sw_pgb_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: tyutyu
--

ALTER SEQUENCE public.sw_pgb_id_seq OWNED BY public.sw_pgb.id;


--
-- TOC entry 4167 (class 2604 OID 19370)
-- Name: pgb_log id; Type: DEFAULT; Schema: pgb; Owner: tyutyu
--

ALTER TABLE ONLY pgb.pgb_log ALTER COLUMN id SET DEFAULT nextval('pgb.pgb_log_id_seq'::regclass);


--
-- TOC entry 4145 (class 2604 OID 18266)
-- Name: ps_notifications id; Type: DEFAULT; Schema: public; Owner: tyutyu
--

ALTER TABLE ONLY public.ps_notifications ALTER COLUMN id SET DEFAULT nextval('public.notify_id_seq'::regclass);


--
-- TOC entry 4155 (class 2604 OID 19213)
-- Name: ps_sw id; Type: DEFAULT; Schema: public; Owner: tyutyu
--

ALTER TABLE ONLY public.ps_sw ALTER COLUMN id SET DEFAULT nextval('public.ps_sw_id_seq'::regclass);


--
-- TOC entry 4156 (class 2604 OID 19235)
-- Name: sw_instance id; Type: DEFAULT; Schema: public; Owner: tyutyu
--

ALTER TABLE ONLY public.sw_instance ALTER COLUMN id SET DEFAULT nextval('public.sw_instance_id_seq'::regclass);


--
-- TOC entry 4162 (class 2604 OID 19317)
-- Name: sw_instance_menu id; Type: DEFAULT; Schema: public; Owner: tyutyu
--

ALTER TABLE ONLY public.sw_instance_menu ALTER COLUMN id SET DEFAULT nextval('public.sw_instance_menu_id_seq'::regclass);


--
-- TOC entry 4166 (class 2604 OID 19349)
-- Name: sw_instance_role_menu id; Type: DEFAULT; Schema: public; Owner: tyutyu
--

ALTER TABLE ONLY public.sw_instance_role_menu ALTER COLUMN id SET DEFAULT nextval('public.sw_instance_role_menu_id_seq'::regclass);


--
-- TOC entry 4161 (class 2604 OID 19308)
-- Name: sw_instance_roles id; Type: DEFAULT; Schema: public; Owner: tyutyu
--

ALTER TABLE ONLY public.sw_instance_roles ALTER COLUMN id SET DEFAULT nextval('public.sw_instance_roles_id_seq'::regclass);


--
-- TOC entry 4165 (class 2604 OID 19338)
-- Name: sw_instance_user id; Type: DEFAULT; Schema: public; Owner: tyutyu
--

ALTER TABLE ONLY public.sw_instance_user ALTER COLUMN id SET DEFAULT nextval('public.sw_instance_user_id_seq'::regclass);


--
-- TOC entry 4160 (class 2604 OID 19261)
-- Name: sw_pgb id; Type: DEFAULT; Schema: public; Owner: tyutyu
--

ALTER TABLE ONLY public.sw_pgb ALTER COLUMN id SET DEFAULT nextval('public.sw_pgb_id_seq'::regclass);


--
-- TOC entry 4368 (class 0 OID 19367)
-- Dependencies: 237
-- Data for Name: pgb_log; Type: TABLE DATA; Schema: pgb; Owner: tyutyu
--

COPY pgb.pgb_log (id, "timestamp", service_name, event_type, database_name, module_name, message, details) FROM stdin;
1	2025-11-02 07:14:34.009651	pgbridge	SERVICE_START	pansoinco pansoinco_suite	\N	pgbridge service connected and initialized	{"version": "1.0.0", "initialized_at": "2025-11-02T07:14:34-05:00"}
2	2025-11-02 07:14:34.014135	pgbridge	MODULE_START	pansoinco pansoinco_suite	pgb_instance_roles	Started module pgb_instance_roles for database pansoinco pansoinco_suite	\N
3	2025-11-02 07:14:34.018719	pgbridge	LISTENER_STARTED	pansoinco pansoinco_suite	\N	Started listening on channel pgb_instance_roles	{"channel": "pgb_instance_roles"}
4	2025-11-02 07:15:11.368965	pgbridge	SERVICE_START	pansoinco pansoinco_suite	\N	pgbridge service connected and initialized	{"version": "1.0.0", "initialized_at": "2025-11-02T07:15:11-05:00"}
5	2025-11-02 07:15:11.379012	pgbridge	MODULE_START	pansoinco pansoinco_suite	pgb_instance_roles	Started module pgb_instance_roles for database pansoinco pansoinco_suite	\N
6	2025-11-02 07:15:11.391449	pgbridge	LISTENER_STARTED	pansoinco pansoinco_suite	\N	Started listening on channel pgb_instance_roles	{"channel": "pgb_instance_roles"}
7	2025-11-02 07:15:11.411312	pgbridge	MODULE_INIT	River river	pgb_notify	Initializing module pgb_notify for database River river	\N
8	2025-11-02 07:15:11.437439	pgbridge	MODULE_START	River river	pgb_notify	Started module pgb_notify for database River river	\N
9	2025-11-02 07:15:11.448354	pgbridge	LISTENER_STARTED	River river	\N	Started listening on channel pgb_notify	{"channel": "pgb_notify"}
10	2025-11-02 07:15:11.453638	pgbridge	MODULE_INIT	River river	pgb_mail	Initializing module pgb_mail for database River river	\N
11	2025-11-02 07:15:11.463026	pgbridge	MODULE_START	River river	pgb_mail	Started module pgb_mail for database River river	\N
12	2025-11-02 07:15:11.473041	pgbridge	LISTENER_STARTED	River river	\N	Started listening on channel pgb_mail	{"channel": "pgb_mail"}
13	2025-11-02 07:15:11.476052	pgbridge	SERVICE_START	\N	\N	pgbridge 1.0.0 started successfully with 2 databases	{"version": "1.0.0", "database_count": 2}
14	2025-11-02 10:27:49.004265	pgbridge	SERVICE_START	pansoinco pansoinco_suite	\N	pgbridge service connected and initialized	{"version": "1.0.0", "initialized_at": "2025-11-02T10:27:49-05:00"}
15	2025-11-02 10:27:49.008385	pgbridge	MODULE_START	pansoinco pansoinco_suite	pgb_instance_roles	Started module pgb_instance_roles for database pansoinco pansoinco_suite	\N
16	2025-11-02 10:27:49.012586	pgbridge	LISTENER_STARTED	pansoinco pansoinco_suite	\N	Started listening on channel pgb_instance_roles	{"channel": "pgb_instance_roles"}
17	2025-11-02 10:27:49.038072	pgbridge	MODULE_INIT	River river	pgb_notify	Initializing module pgb_notify for database River river	\N
18	2025-11-02 10:27:49.06209	pgbridge	MODULE_START	River river	pgb_notify	Started module pgb_notify for database River river	\N
19	2025-11-02 10:27:49.068254	pgbridge	LISTENER_STARTED	River river	\N	Started listening on channel pgb_notify	{"channel": "pgb_notify"}
20	2025-11-02 10:27:49.074401	pgbridge	MODULE_INIT	River river	pgb_mail	Initializing module pgb_mail for database River river	\N
21	2025-11-02 10:27:49.080942	pgbridge	MODULE_START	River river	pgb_mail	Started module pgb_mail for database River river	\N
22	2025-11-02 10:27:49.094773	pgbridge	LISTENER_STARTED	River river	\N	Started listening on channel pgb_mail	{"channel": "pgb_mail"}
23	2025-11-02 10:27:49.097748	pgbridge	SERVICE_START	\N	\N	pgbridge 1.0.0 started successfully with 2 databases	{"version": "1.0.0", "database_count": 2}
24	2025-11-02 10:29:40.86737	pgbridge	SERVICE_START	pansoinco pansoinco_suite	\N	pgbridge service connected and initialized	{"version": "1.0.0", "initialized_at": "2025-11-02T10:29:40-05:00"}
25	2025-11-02 10:29:40.871244	pgbridge	MODULE_START	pansoinco pansoinco_suite	pgb_instance_roles	Started module pgb_instance_roles for database pansoinco pansoinco_suite	\N
26	2025-11-02 10:29:40.8847	pgbridge	LISTENER_STARTED	pansoinco pansoinco_suite	\N	Started listening on channel pgb_instance_roles	{"channel": "pgb_instance_roles"}
27	2025-11-02 10:29:40.905129	pgbridge	MODULE_INIT	River river	pgb_notify	Initializing module pgb_notify for database River river	\N
28	2025-11-02 10:29:40.935506	pgbridge	MODULE_START	River river	pgb_notify	Started module pgb_notify for database River river	\N
29	2025-11-02 10:29:40.942176	pgbridge	LISTENER_STARTED	River river	\N	Started listening on channel pgb_notify	{"channel": "pgb_notify"}
30	2025-11-02 10:29:40.945955	pgbridge	MODULE_INIT	River river	pgb_mail	Initializing module pgb_mail for database River river	\N
31	2025-11-02 10:29:40.956155	pgbridge	MODULE_START	River river	pgb_mail	Started module pgb_mail for database River river	\N
32	2025-11-02 10:29:40.979699	pgbridge	LISTENER_STARTED	River river	\N	Started listening on channel pgb_mail	{"channel": "pgb_mail"}
33	2025-11-02 10:29:40.982721	pgbridge	SERVICE_START	\N	\N	pgbridge 1.0.0 started successfully with 2 databases	{"version": "1.0.0", "database_count": 2}
34	2025-11-02 10:29:45.824033	pgbridge	LISTENER_ERROR	pansoinco pansoinco_suite	\N	Listener error on channel pgb_instance_roles: read tcp [::1]:43946->[::1]:5432: use of closed network connection	{"error": "read tcp [::1]:43946->[::1]:5432: use of closed network connection", "channel": "pgb_instance_roles"}
35	2025-11-02 10:30:00.664676	pgbridge	SERVICE_START	pansoinco pansoinco_suite	\N	pgbridge service connected and initialized	{"version": "1.0.0", "initialized_at": "2025-11-02T10:30:00-05:00"}
36	2025-11-02 10:30:00.669071	pgbridge	MODULE_START	pansoinco pansoinco_suite	pgb_instance_roles	Started module pgb_instance_roles for database pansoinco pansoinco_suite	\N
37	2025-11-02 10:30:00.67503	pgbridge	LISTENER_STARTED	pansoinco pansoinco_suite	\N	Started listening on channel pgb_instance_roles	{"channel": "pgb_instance_roles"}
38	2025-11-02 10:30:00.692291	pgbridge	MODULE_INIT	River river	pgb_notify	Initializing module pgb_notify for database River river	\N
39	2025-11-02 10:30:00.71735	pgbridge	MODULE_START	River river	pgb_notify	Started module pgb_notify for database River river	\N
40	2025-11-02 10:30:00.730176	pgbridge	LISTENER_STARTED	River river	\N	Started listening on channel pgb_notify	{"channel": "pgb_notify"}
41	2025-11-02 10:30:00.734899	pgbridge	MODULE_INIT	River river	pgb_mail	Initializing module pgb_mail for database River river	\N
42	2025-11-02 10:30:00.740088	pgbridge	MODULE_START	River river	pgb_mail	Started module pgb_mail for database River river	\N
43	2025-11-02 10:30:00.746685	pgbridge	LISTENER_STARTED	River river	\N	Started listening on channel pgb_mail	{"channel": "pgb_mail"}
44	2025-11-02 10:30:00.749193	pgbridge	SERVICE_START	\N	\N	pgbridge 1.0.0 started successfully with 2 databases	{"version": "1.0.0", "database_count": 2}
45	2025-11-02 10:30:05.631066	pgbridge	LISTENER_ERROR	pansoinco pansoinco_suite	\N	Listener error on channel pgb_instance_roles: read tcp [::1]:53174->[::1]:5432: use of closed network connection	{"error": "read tcp [::1]:53174->[::1]:5432: use of closed network connection", "channel": "pgb_instance_roles"}
\.


--
-- TOC entry 4358 (class 0 OID 19291)
-- Dependencies: 227
-- Data for Name: ps_auth_token; Type: TABLE DATA; Schema: public; Owner: tyutyu
--

COPY public.ps_auth_token (email, access_token, issued, expires, is_expired) FROM stdin;
\.


--
-- TOC entry 4350 (class 0 OID 18263)
-- Dependencies: 219
-- Data for Name: ps_notifications; Type: TABLE DATA; Schema: public; Owner: tyutyu
--

COPY public.ps_notifications (id, user_email, received_ts, sender_db, message, message_link, is_seen, seen_ts, criticality, original_id) FROM stdin;
1	botondzalai@pansoinco.com	2025-10-31 19:06:20.126488	river	This is our first distributed message		t	2025-11-01 00:07:18.079431	2	1
2	botond@pansoinco.com	2025-11-02 06:57:44.304511	river	Testing second time	telex.hu	f	\N	1	2
\.


--
-- TOC entry 4353 (class 0 OID 19210)
-- Dependencies: 222
-- Data for Name: ps_sw; Type: TABLE DATA; Schema: public; Owner: tyutyu
--

COPY public.ps_sw (id, short_name, long_name) FROM stdin;
1	River	Test database River
2	pansoinco	PanSoinco Suite
\.


--
-- TOC entry 4351 (class 0 OID 19188)
-- Dependencies: 220
-- Data for Name: ps_users; Type: TABLE DATA; Schema: public; Owner: tyutyu
--

COPY public.ps_users (email, first_name, last_name, password, ps_superuser, created_by, created_at, last_login, parent_user, totp_enabled, totp_enforced, totp_secret, totp_backup, oauth_enabled, oauth_enforced, oauth_provider, oauth_userid, oauth_access_token, oauth_refresh_token, oauth_token_expires, is_user_disabled) FROM stdin;
\.


--
-- TOC entry 4355 (class 0 OID 19232)
-- Dependencies: 224
-- Data for Name: sw_instance; Type: TABLE DATA; Schema: public; Owner: tyutyu
--

COPY public.sw_instance (id, sw_id, instance_name, instance_type, db_name, db_host, db_port, db_owner, db_password, db_connection_string, rel_docroot, uri) FROM stdin;
3	1	Danube	development	river	localhost	5432	tyutyu	801031	postgres://tyutyu:'801031'@localhost:5432/river	river	localhost/danube/
4	2	suite	production	pansoinco_suite	localhost	5432	tyutyu	801031	postgres://tyutyu:'801031'@localhost:5432/pansoinco_suite	/	https://suite.example.com
5	2	test	development	pansoinco_suite	localhost	5432	tyutyu	801031	postgres://tyutyu:'801031'@localhost:5432/pansoinco_suite	/	https://test.example.com
\.


--
-- TOC entry 4362 (class 0 OID 19314)
-- Dependencies: 231
-- Data for Name: sw_instance_menu; Type: TABLE DATA; Schema: public; Owner: tyutyu
--

COPY public.sw_instance_menu (id, sw_instance_id, menu_title, is_uri, superior_menu, menu_icon, filename, get_args, ordering) FROM stdin;
\.


--
-- TOC entry 4366 (class 0 OID 19346)
-- Dependencies: 235
-- Data for Name: sw_instance_role_menu; Type: TABLE DATA; Schema: public; Owner: tyutyu
--

COPY public.sw_instance_role_menu (id, instancemenu_id, instancerole_id) FROM stdin;
\.


--
-- TOC entry 4360 (class 0 OID 19305)
-- Dependencies: 229
-- Data for Name: sw_instance_roles; Type: TABLE DATA; Schema: public; Owner: tyutyu
--

COPY public.sw_instance_roles (id, instance_id, role, role_description) FROM stdin;
1	3	boat_owners	\N
2	3	riverside	\N
3	3	tyutyu	\N
4	5	boat_owners	\N
5	5	riverside	\N
6	5	tyutyu	\N
\.


--
-- TOC entry 4364 (class 0 OID 19335)
-- Dependencies: 233
-- Data for Name: sw_instance_user; Type: TABLE DATA; Schema: public; Owner: tyutyu
--

COPY public.sw_instance_user (id, user_email, instance_id, role_id) FROM stdin;
\.


--
-- TOC entry 4357 (class 0 OID 19258)
-- Dependencies: 226
-- Data for Name: sw_pgb; Type: TABLE DATA; Schema: public; Owner: tyutyu
--

COPY public.sw_pgb (id, sw_instance_id, pgb_role, pgb_password, pgb_services, connection_string) FROM stdin;
2	3	tyutyu	801031	{pgb_notify,pgb_mail}	postgres://tyutyu:'801031'@localhost:5432/river
3	4	tyutyu	801031	{pgb_instance_roles}	postgres://tyutyu:'801031'@localhost:5432/suite
\.


--
-- TOC entry 4395 (class 0 OID 0)
-- Dependencies: 236
-- Name: pgb_log_id_seq; Type: SEQUENCE SET; Schema: pgb; Owner: tyutyu
--

SELECT pg_catalog.setval('pgb.pgb_log_id_seq', 45, true);


--
-- TOC entry 4396 (class 0 OID 0)
-- Dependencies: 218
-- Name: notify_id_seq; Type: SEQUENCE SET; Schema: public; Owner: tyutyu
--

SELECT pg_catalog.setval('public.notify_id_seq', 2, true);


--
-- TOC entry 4397 (class 0 OID 0)
-- Dependencies: 221
-- Name: ps_sw_id_seq; Type: SEQUENCE SET; Schema: public; Owner: tyutyu
--

SELECT pg_catalog.setval('public.ps_sw_id_seq', 2, true);


--
-- TOC entry 4398 (class 0 OID 0)
-- Dependencies: 223
-- Name: sw_instance_id_seq; Type: SEQUENCE SET; Schema: public; Owner: tyutyu
--

SELECT pg_catalog.setval('public.sw_instance_id_seq', 5, true);


--
-- TOC entry 4399 (class 0 OID 0)
-- Dependencies: 230
-- Name: sw_instance_menu_id_seq; Type: SEQUENCE SET; Schema: public; Owner: tyutyu
--

SELECT pg_catalog.setval('public.sw_instance_menu_id_seq', 1, false);


--
-- TOC entry 4400 (class 0 OID 0)
-- Dependencies: 234
-- Name: sw_instance_role_menu_id_seq; Type: SEQUENCE SET; Schema: public; Owner: tyutyu
--

SELECT pg_catalog.setval('public.sw_instance_role_menu_id_seq', 1, false);


--
-- TOC entry 4401 (class 0 OID 0)
-- Dependencies: 228
-- Name: sw_instance_roles_id_seq; Type: SEQUENCE SET; Schema: public; Owner: tyutyu
--

SELECT pg_catalog.setval('public.sw_instance_roles_id_seq', 9, true);


--
-- TOC entry 4402 (class 0 OID 0)
-- Dependencies: 232
-- Name: sw_instance_user_id_seq; Type: SEQUENCE SET; Schema: public; Owner: tyutyu
--

SELECT pg_catalog.setval('public.sw_instance_user_id_seq', 1, false);


--
-- TOC entry 4403 (class 0 OID 0)
-- Dependencies: 225
-- Name: sw_pgb_id_seq; Type: SEQUENCE SET; Schema: public; Owner: tyutyu
--

SELECT pg_catalog.setval('public.sw_pgb_id_seq', 3, true);


--
-- TOC entry 4199 (class 2606 OID 19376)
-- Name: pgb_log pgb_log_pkey; Type: CONSTRAINT; Schema: pgb; Owner: tyutyu
--

ALTER TABLE ONLY pgb.pgb_log
    ADD CONSTRAINT pgb_log_pkey PRIMARY KEY (id);


--
-- TOC entry 4171 (class 2606 OID 18271)
-- Name: ps_notifications notify_pk; Type: CONSTRAINT; Schema: public; Owner: tyutyu
--

ALTER TABLE ONLY public.ps_notifications
    ADD CONSTRAINT notify_pk PRIMARY KEY (id);


--
-- TOC entry 4175 (class 2606 OID 19217)
-- Name: ps_sw ps_sw_pk; Type: CONSTRAINT; Schema: public; Owner: tyutyu
--

ALTER TABLE ONLY public.ps_sw
    ADD CONSTRAINT ps_sw_pk PRIMARY KEY (id);


--
-- TOC entry 4177 (class 2606 OID 19219)
-- Name: ps_sw ps_sw_unique; Type: CONSTRAINT; Schema: public; Owner: tyutyu
--

ALTER TABLE ONLY public.ps_sw
    ADD CONSTRAINT ps_sw_unique UNIQUE (short_name);


--
-- TOC entry 4173 (class 2606 OID 19327)
-- Name: ps_users ps_users_pk; Type: CONSTRAINT; Schema: public; Owner: tyutyu
--

ALTER TABLE ONLY public.ps_users
    ADD CONSTRAINT ps_users_pk PRIMARY KEY (email);


--
-- TOC entry 4179 (class 2606 OID 19242)
-- Name: sw_instance pssw_instance_pk; Type: CONSTRAINT; Schema: public; Owner: tyutyu
--

ALTER TABLE ONLY public.sw_instance
    ADD CONSTRAINT pssw_instance_pk PRIMARY KEY (id);


--
-- TOC entry 4189 (class 2606 OID 19322)
-- Name: sw_instance_menu sw_instance_menu_pk; Type: CONSTRAINT; Schema: public; Owner: tyutyu
--

ALTER TABLE ONLY public.sw_instance_menu
    ADD CONSTRAINT sw_instance_menu_pk PRIMARY KEY (id);


--
-- TOC entry 4195 (class 2606 OID 19351)
-- Name: sw_instance_role_menu sw_instance_role_menu_pk; Type: CONSTRAINT; Schema: public; Owner: tyutyu
--

ALTER TABLE ONLY public.sw_instance_role_menu
    ADD CONSTRAINT sw_instance_role_menu_pk PRIMARY KEY (id);


--
-- TOC entry 4185 (class 2606 OID 19312)
-- Name: sw_instance_roles sw_instance_roles_pk; Type: CONSTRAINT; Schema: public; Owner: tyutyu
--

ALTER TABLE ONLY public.sw_instance_roles
    ADD CONSTRAINT sw_instance_roles_pk PRIMARY KEY (id);


--
-- TOC entry 4187 (class 2606 OID 19361)
-- Name: sw_instance_roles sw_instance_roles_unique; Type: CONSTRAINT; Schema: public; Owner: tyutyu
--

ALTER TABLE ONLY public.sw_instance_roles
    ADD CONSTRAINT sw_instance_roles_unique UNIQUE (instance_id, role);


--
-- TOC entry 4191 (class 2606 OID 19342)
-- Name: sw_instance_user sw_instance_user_pk; Type: CONSTRAINT; Schema: public; Owner: tyutyu
--

ALTER TABLE ONLY public.sw_instance_user
    ADD CONSTRAINT sw_instance_user_pk PRIMARY KEY (id);


--
-- TOC entry 4193 (class 2606 OID 19344)
-- Name: sw_instance_user sw_instance_user_unique; Type: CONSTRAINT; Schema: public; Owner: tyutyu
--

ALTER TABLE ONLY public.sw_instance_user
    ADD CONSTRAINT sw_instance_user_unique UNIQUE (user_email, instance_id);


--
-- TOC entry 4181 (class 2606 OID 19265)
-- Name: sw_pgb sw_pgb_pk; Type: CONSTRAINT; Schema: public; Owner: tyutyu
--

ALTER TABLE ONLY public.sw_pgb
    ADD CONSTRAINT sw_pgb_pk PRIMARY KEY (id);


--
-- TOC entry 4183 (class 2606 OID 19284)
-- Name: sw_pgb sw_pgb_unique; Type: CONSTRAINT; Schema: public; Owner: tyutyu
--

ALTER TABLE ONLY public.sw_pgb
    ADD CONSTRAINT sw_pgb_unique UNIQUE (sw_instance_id, pgb_role);


--
-- TOC entry 4196 (class 1259 OID 19378)
-- Name: idx_pgb_log_event_type; Type: INDEX; Schema: pgb; Owner: tyutyu
--

CREATE INDEX idx_pgb_log_event_type ON pgb.pgb_log USING btree (event_type);


--
-- TOC entry 4197 (class 1259 OID 19377)
-- Name: idx_pgb_log_timestamp; Type: INDEX; Schema: pgb; Owner: tyutyu
--

CREATE INDEX idx_pgb_log_timestamp ON pgb.pgb_log USING btree ("timestamp");


--
-- TOC entry 4201 (class 2620 OID 19290)
-- Name: sw_instance B1_connection_string; Type: TRIGGER; Schema: public; Owner: tyutyu
--

CREATE TRIGGER "B1_connection_string" BEFORE INSERT OR UPDATE ON public.sw_instance FOR EACH ROW EXECUTE FUNCTION public.trg_sw_instance_connection_string();


--
-- TOC entry 4203 (class 2620 OID 19286)
-- Name: sw_pgb B1_connection_string; Type: TRIGGER; Schema: public; Owner: tyutyu
--

CREATE TRIGGER "B1_connection_string" BEFORE INSERT OR UPDATE ON public.sw_pgb FOR EACH ROW EXECUTE FUNCTION public.trg_pgb_connection_string();


--
-- TOC entry 4202 (class 2620 OID 19363)
-- Name: sw_instance s01_instance_roles_notify; Type: TRIGGER; Schema: public; Owner: tyutyu
--

CREATE TRIGGER s01_instance_roles_notify AFTER INSERT ON public.sw_instance FOR EACH ROW EXECUTE FUNCTION public.trg_instance_roles_notify();


--
-- TOC entry 4200 (class 2606 OID 19278)
-- Name: sw_pgb sw_pgb_sw_instance_fk; Type: FK CONSTRAINT; Schema: public; Owner: tyutyu
--

ALTER TABLE ONLY public.sw_pgb
    ADD CONSTRAINT sw_pgb_sw_instance_fk FOREIGN KEY (sw_instance_id) REFERENCES public.sw_instance(id) ON DELETE CASCADE;


-- Completed on 2025-11-02 11:19:30 EST

--
-- PostgreSQL database dump complete
--

--
-- Database "river" dump
--

--
-- PostgreSQL database dump
--

-- Dumped from database version 17.5
-- Dumped by pg_dump version 17.5

-- Started on 2025-11-02 11:19:30 EST

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- TOC entry 4300 (class 1262 OID 16399)
-- Name: river; Type: DATABASE; Schema: -; Owner: tyutyu
--

CREATE DATABASE river WITH TEMPLATE = template0 ENCODING = 'UTF8' LOCALE_PROVIDER = libc LOCALE = 'en_US.UTF-8';


ALTER DATABASE river OWNER TO tyutyu;

\connect river

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- TOC entry 5 (class 2615 OID 19000)
-- Name: pgb; Type: SCHEMA; Schema: -; Owner: tyutyu
--

CREATE SCHEMA pgb;


ALTER SCHEMA pgb OWNER TO tyutyu;

--
-- TOC entry 226 (class 1255 OID 19353)
-- Name: trg_pgb_send_notification(); Type: FUNCTION; Schema: pgb; Owner: tyutyu
--

CREATE FUNCTION pgb.trg_pgb_send_notification() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
		BEGIN
			-- Send NOTIFY with the new notification ID
			PERFORM pg_notify('pgb_notify', NEW.id::text);

			-- Mark as sent (will be handled by pgbridge)
			-- Note: We don't set is_sent here as pgbridge will do it after forwarding

			RETURN NEW;
		END;
		$$;


ALTER FUNCTION pgb.trg_pgb_send_notification() OWNER TO tyutyu;

--
-- TOC entry 4302 (class 0 OID 0)
-- Dependencies: 226
-- Name: FUNCTION trg_pgb_send_notification(); Type: COMMENT; Schema: pgb; Owner: tyutyu
--

COMMENT ON FUNCTION pgb.trg_pgb_send_notification() IS 'Automatically sends NOTIFY when notification is inserted';


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- TOC entry 219 (class 1259 OID 19002)
-- Name: pgb_log; Type: TABLE; Schema: pgb; Owner: tyutyu
--

CREATE TABLE pgb.pgb_log (
    id integer NOT NULL,
    "timestamp" timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    service_name character varying(50) DEFAULT 'pgbridge'::character varying,
    event_type character varying(50) NOT NULL,
    database_name character varying(100),
    module_name character varying(50),
    message text,
    details jsonb
);


ALTER TABLE pgb.pgb_log OWNER TO tyutyu;

--
-- TOC entry 218 (class 1259 OID 19001)
-- Name: pgb_log_id_seq; Type: SEQUENCE; Schema: pgb; Owner: tyutyu
--

CREATE SEQUENCE pgb.pgb_log_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE pgb.pgb_log_id_seq OWNER TO tyutyu;

--
-- TOC entry 4303 (class 0 OID 0)
-- Dependencies: 218
-- Name: pgb_log_id_seq; Type: SEQUENCE OWNED BY; Schema: pgb; Owner: tyutyu
--

ALTER SEQUENCE pgb.pgb_log_id_seq OWNED BY pgb.pgb_log.id;


--
-- TOC entry 223 (class 1259 OID 19030)
-- Name: pgb_mail; Type: TABLE; Schema: pgb; Owner: tyutyu
--

CREATE TABLE pgb.pgb_mail (
    id integer NOT NULL,
    mail_setting_id integer NOT NULL,
    header_from character varying(255) NOT NULL,
    header_to text NOT NULL,
    header_cc text,
    header_bcc text,
    subject character varying(998) NOT NULL,
    body_text text NOT NULL,
    is_sent boolean DEFAULT false,
    sent_ts timestamp without time zone,
    error_message text,
    retry_count integer DEFAULT 0,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT valid_email_to CHECK ((header_to <> ''::text)),
    CONSTRAINT valid_subject CHECK (((subject)::text <> ''::text))
);


ALTER TABLE pgb.pgb_mail OWNER TO tyutyu;

--
-- TOC entry 4304 (class 0 OID 0)
-- Dependencies: 223
-- Name: TABLE pgb_mail; Type: COMMENT; Schema: pgb; Owner: tyutyu
--

COMMENT ON TABLE pgb.pgb_mail IS 'Email queue for pgbridge mail module';


--
-- TOC entry 4305 (class 0 OID 0)
-- Dependencies: 223
-- Name: COLUMN pgb_mail.header_to; Type: COMMENT; Schema: pgb; Owner: tyutyu
--

COMMENT ON COLUMN pgb.pgb_mail.header_to IS 'Comma-separated list of recipient email addresses';


--
-- TOC entry 4306 (class 0 OID 0)
-- Dependencies: 223
-- Name: COLUMN pgb_mail.header_cc; Type: COMMENT; Schema: pgb; Owner: tyutyu
--

COMMENT ON COLUMN pgb.pgb_mail.header_cc IS 'Comma-separated list of CC email addresses';


--
-- TOC entry 4307 (class 0 OID 0)
-- Dependencies: 223
-- Name: COLUMN pgb_mail.header_bcc; Type: COMMENT; Schema: pgb; Owner: tyutyu
--

COMMENT ON COLUMN pgb.pgb_mail.header_bcc IS 'Comma-separated list of BCC email addresses';


--
-- TOC entry 4308 (class 0 OID 0)
-- Dependencies: 223
-- Name: COLUMN pgb_mail.retry_count; Type: COMMENT; Schema: pgb; Owner: tyutyu
--

COMMENT ON COLUMN pgb.pgb_mail.retry_count IS 'Number of send attempts';


--
-- TOC entry 222 (class 1259 OID 19029)
-- Name: pgb_mail_id_seq; Type: SEQUENCE; Schema: pgb; Owner: tyutyu
--

CREATE SEQUENCE pgb.pgb_mail_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE pgb.pgb_mail_id_seq OWNER TO tyutyu;

--
-- TOC entry 4309 (class 0 OID 0)
-- Dependencies: 222
-- Name: pgb_mail_id_seq; Type: SEQUENCE OWNED BY; Schema: pgb; Owner: tyutyu
--

ALTER SEQUENCE pgb.pgb_mail_id_seq OWNED BY pgb.pgb_mail.id;


--
-- TOC entry 221 (class 1259 OID 19015)
-- Name: pgb_mail_settings; Type: TABLE; Schema: pgb; Owner: tyutyu
--

CREATE TABLE pgb.pgb_mail_settings (
    id integer NOT NULL,
    smtp_server character varying(255) NOT NULL,
    smtp_port integer DEFAULT 587 NOT NULL,
    is_tls boolean DEFAULT true,
    is_ssl boolean DEFAULT false,
    smtp_user character varying(255),
    smtp_password character varying(255),
    smtp_token text,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT valid_port CHECK (((smtp_port > 0) AND (smtp_port <= 65535)))
);


ALTER TABLE pgb.pgb_mail_settings OWNER TO tyutyu;

--
-- TOC entry 4310 (class 0 OID 0)
-- Dependencies: 221
-- Name: TABLE pgb_mail_settings; Type: COMMENT; Schema: pgb; Owner: tyutyu
--

COMMENT ON TABLE pgb.pgb_mail_settings IS 'SMTP server configuration for pgbridge mail module';


--
-- TOC entry 4311 (class 0 OID 0)
-- Dependencies: 221
-- Name: COLUMN pgb_mail_settings.is_tls; Type: COMMENT; Schema: pgb; Owner: tyutyu
--

COMMENT ON COLUMN pgb.pgb_mail_settings.is_tls IS 'Use STARTTLS encryption';


--
-- TOC entry 4312 (class 0 OID 0)
-- Dependencies: 221
-- Name: COLUMN pgb_mail_settings.is_ssl; Type: COMMENT; Schema: pgb; Owner: tyutyu
--

COMMENT ON COLUMN pgb.pgb_mail_settings.is_ssl IS 'Use SSL/TLS from the start';


--
-- TOC entry 4313 (class 0 OID 0)
-- Dependencies: 221
-- Name: COLUMN pgb_mail_settings.smtp_token; Type: COMMENT; Schema: pgb; Owner: tyutyu
--

COMMENT ON COLUMN pgb.pgb_mail_settings.smtp_token IS 'API token for token-based authentication';


--
-- TOC entry 220 (class 1259 OID 19014)
-- Name: pgb_mail_settings_id_seq; Type: SEQUENCE; Schema: pgb; Owner: tyutyu
--

CREATE SEQUENCE pgb.pgb_mail_settings_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE pgb.pgb_mail_settings_id_seq OWNER TO tyutyu;

--
-- TOC entry 4314 (class 0 OID 0)
-- Dependencies: 220
-- Name: pgb_mail_settings_id_seq; Type: SEQUENCE OWNED BY; Schema: pgb; Owner: tyutyu
--

ALTER SEQUENCE pgb.pgb_mail_settings_id_seq OWNED BY pgb.pgb_mail_settings.id;


--
-- TOC entry 225 (class 1259 OID 19053)
-- Name: pgb_notify; Type: TABLE; Schema: pgb; Owner: tyutyu
--

CREATE TABLE pgb.pgb_notify (
    id integer NOT NULL,
    user_email character varying(255) NOT NULL,
    sender_db character varying(100) NOT NULL,
    message text,
    message_link character varying(500),
    criticality smallint DEFAULT 1 NOT NULL,
    is_sent boolean DEFAULT false NOT NULL,
    sent_ts timestamp without time zone,
    is_seen boolean DEFAULT false NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT valid_criticality CHECK (((criticality >= 1) AND (criticality <= 5)))
);


ALTER TABLE pgb.pgb_notify OWNER TO tyutyu;

--
-- TOC entry 4315 (class 0 OID 0)
-- Dependencies: 225
-- Name: TABLE pgb_notify; Type: COMMENT; Schema: pgb; Owner: tyutyu
--

COMMENT ON TABLE pgb.pgb_notify IS 'Notification queue for pgbridge notify module';


--
-- TOC entry 4316 (class 0 OID 0)
-- Dependencies: 225
-- Name: COLUMN pgb_notify.criticality; Type: COMMENT; Schema: pgb; Owner: tyutyu
--

COMMENT ON COLUMN pgb.pgb_notify.criticality IS 'Criticality level: 1=Info, 2=Low, 3=Medium, 4=High, 5=Critical';


--
-- TOC entry 4317 (class 0 OID 0)
-- Dependencies: 225
-- Name: COLUMN pgb_notify.is_sent; Type: COMMENT; Schema: pgb; Owner: tyutyu
--

COMMENT ON COLUMN pgb.pgb_notify.is_sent IS 'Whether notification was sent to central database';


--
-- TOC entry 4318 (class 0 OID 0)
-- Dependencies: 225
-- Name: COLUMN pgb_notify.is_seen; Type: COMMENT; Schema: pgb; Owner: tyutyu
--

COMMENT ON COLUMN pgb.pgb_notify.is_seen IS 'Whether notification was marked as seen in central database';


--
-- TOC entry 224 (class 1259 OID 19052)
-- Name: pgb_notify_id_seq; Type: SEQUENCE; Schema: pgb; Owner: tyutyu
--

CREATE SEQUENCE pgb.pgb_notify_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE pgb.pgb_notify_id_seq OWNER TO tyutyu;

--
-- TOC entry 4319 (class 0 OID 0)
-- Dependencies: 224
-- Name: pgb_notify_id_seq; Type: SEQUENCE OWNED BY; Schema: pgb; Owner: tyutyu
--

ALTER SEQUENCE pgb.pgb_notify_id_seq OWNED BY pgb.pgb_notify.id;


--
-- TOC entry 4098 (class 2604 OID 19005)
-- Name: pgb_log id; Type: DEFAULT; Schema: pgb; Owner: tyutyu
--

ALTER TABLE ONLY pgb.pgb_log ALTER COLUMN id SET DEFAULT nextval('pgb.pgb_log_id_seq'::regclass);


--
-- TOC entry 4107 (class 2604 OID 19033)
-- Name: pgb_mail id; Type: DEFAULT; Schema: pgb; Owner: tyutyu
--

ALTER TABLE ONLY pgb.pgb_mail ALTER COLUMN id SET DEFAULT nextval('pgb.pgb_mail_id_seq'::regclass);


--
-- TOC entry 4101 (class 2604 OID 19018)
-- Name: pgb_mail_settings id; Type: DEFAULT; Schema: pgb; Owner: tyutyu
--

ALTER TABLE ONLY pgb.pgb_mail_settings ALTER COLUMN id SET DEFAULT nextval('pgb.pgb_mail_settings_id_seq'::regclass);


--
-- TOC entry 4112 (class 2604 OID 19056)
-- Name: pgb_notify id; Type: DEFAULT; Schema: pgb; Owner: tyutyu
--

ALTER TABLE ONLY pgb.pgb_notify ALTER COLUMN id SET DEFAULT nextval('pgb.pgb_notify_id_seq'::regclass);


--
-- TOC entry 4288 (class 0 OID 19002)
-- Dependencies: 219
-- Data for Name: pgb_log; Type: TABLE DATA; Schema: pgb; Owner: tyutyu
--

COPY pgb.pgb_log (id, "timestamp", service_name, event_type, database_name, module_name, message, details) FROM stdin;
1	2025-10-31 17:52:37.342049	pgbridge	SERVICE_START	River	\N	pgbridge service connected and initialized	{"version": "1.0.0", "initialized_at": "2025-10-31T17:52:37-04:00"}
2	2025-10-31 17:52:37.398788	pgbridge	MODULE_INIT	River	pgb_mail	Initializing module pgb_mail for database River	\N
3	2025-10-31 17:52:37.670948	pgbridge	MODULE_START	River	pgb_mail	Started module pgb_mail for database River	\N
4	2025-10-31 17:55:06.685194	pgbridge	SERVICE_START	River	\N	pgbridge service connected and initialized	{"version": "1.0.0", "initialized_at": "2025-10-31T17:55:06-04:00"}
5	2025-10-31 17:55:06.714238	pgbridge	MODULE_INIT	River	pgb_mail	Initializing module pgb_mail for database River	\N
6	2025-10-31 17:55:06.721958	pgbridge	MODULE_START	River	pgb_mail	Started module pgb_mail for database River	\N
7	2025-10-31 17:55:30.363696	pgbridge	SERVICE_START	River	\N	pgbridge service connected and initialized	{"version": "1.0.0", "initialized_at": "2025-10-31T17:55:30-04:00"}
8	2025-10-31 17:55:30.369006	pgbridge	MODULE_INIT	River	pgb_mail	Initializing module pgb_mail for database River	\N
9	2025-10-31 17:55:30.374196	pgbridge	MODULE_START	River	pgb_mail	Started module pgb_mail for database River	\N
10	2025-10-31 17:55:30.381362	pgbridge	LISTENER_STARTED	River	\N	Started listening on channel pgb_mail	{"channel": "pgb_mail"}
11	2025-10-31 17:55:30.403014	pgbridge	MODULE_INIT	River	pgb_notify	Initializing module pgb_notify for database River	\N
12	2025-10-31 17:55:30.77146	pgbridge	MODULE_START	River	pgb_notify	Started module pgb_notify for database River	\N
13	2025-10-31 17:55:30.799444	pgbridge	LISTENER_STARTED	River	\N	Started listening on channel pgb_notify	{"channel": "pgb_notify"}
14	2025-10-31 17:55:31.426069	pgbridge	MODULE_INIT	Troubled Water	pgb_mail	Initializing module pgb_mail for database Troubled Water	\N
15	2025-10-31 17:55:31.744141	pgbridge	MODULE_START	Troubled Water	pgb_mail	Started module pgb_mail for database Troubled Water	\N
16	2025-10-31 17:55:31.79095	pgbridge	LISTENER_STARTED	Troubled Water	\N	Started listening on channel pgb_mail	{"channel": "pgb_mail"}
17	2025-10-31 17:55:31.794569	pgbridge	MODULE_INIT	Troubled Water	pgb_notify	Initializing module pgb_notify for database Troubled Water	\N
18	2025-10-31 17:55:31.818844	pgbridge	MODULE_START	Troubled Water	pgb_notify	Started module pgb_notify for database Troubled Water	\N
19	2025-10-31 17:55:31.829085	pgbridge	LISTENER_STARTED	Troubled Water	\N	Started listening on channel pgb_notify	{"channel": "pgb_notify"}
20	2025-10-31 17:55:31.831596	pgbridge	SERVICE_START	\N	\N	pgbridge 1.0.0 started successfully with 2 databases	{"version": "1.0.0", "database_count": 2}
21	2025-10-31 17:55:40.390715	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
22	2025-10-31 17:55:40.780589	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
23	2025-10-31 17:55:41.792908	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
24	2025-10-31 17:55:41.829693	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
25	2025-10-31 17:55:50.398675	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
26	2025-10-31 17:55:51.2047	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
27	2025-10-31 17:55:52.186389	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
28	2025-10-31 17:55:52.189731	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
29	2025-10-31 17:56:00.40224	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
30	2025-10-31 17:56:01.346111	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
31	2025-10-31 17:56:02.188404	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
32	2025-10-31 17:56:02.192713	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
33	2025-10-31 17:56:10.407466	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
34	2025-10-31 17:56:11.350243	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
35	2025-10-31 17:56:12.190201	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
36	2025-10-31 17:56:12.201676	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
37	2025-10-31 17:56:20.412404	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
38	2025-10-31 17:56:21.350575	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
39	2025-10-31 17:56:22.191213	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
40	2025-10-31 17:56:22.19559	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
41	2025-10-31 17:56:30.412601	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
42	2025-10-31 17:56:31.35141	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
43	2025-10-31 17:56:32.259627	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
44	2025-10-31 17:56:32.264604	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
45	2025-10-31 17:56:40.425318	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
46	2025-10-31 17:56:41.351694	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
47	2025-10-31 17:56:42.260576	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
48	2025-10-31 17:56:42.309298	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
49	2025-10-31 17:56:50.431892	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
50	2025-10-31 17:56:51.352649	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
51	2025-10-31 17:56:52.261718	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
52	2025-10-31 17:56:52.265865	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
53	2025-10-31 17:57:00.432258	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
54	2025-10-31 17:57:01.356244	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
55	2025-10-31 17:57:02.26201	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
56	2025-10-31 17:57:02.2672	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
57	2025-10-31 17:57:10.433496	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
58	2025-10-31 17:57:11.355979	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
59	2025-10-31 17:57:12.263999	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
60	2025-10-31 17:57:12.269379	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
61	2025-10-31 17:57:20.439125	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
62	2025-10-31 17:57:21.356436	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
63	2025-10-31 17:57:22.265596	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
64	2025-10-31 17:57:22.271233	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
65	2025-10-31 17:57:30.441439	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
66	2025-10-31 17:57:31.426429	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
67	2025-10-31 17:57:32.266124	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
68	2025-10-31 17:57:32.270913	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
69	2025-10-31 17:57:40.442474	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
70	2025-10-31 17:57:41.426974	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
71	2025-10-31 17:57:42.267869	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
72	2025-10-31 17:57:42.272582	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
73	2025-10-31 17:57:50.442594	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
74	2025-10-31 17:57:51.427634	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
75	2025-10-31 17:57:52.269641	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
76	2025-10-31 17:57:52.274206	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
77	2025-10-31 17:58:00.443763	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
78	2025-10-31 17:58:01.42727	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
79	2025-10-31 17:58:02.46541	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
80	2025-10-31 17:58:02.487256	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
81	2025-10-31 17:58:10.445726	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
82	2025-10-31 17:58:11.428208	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
83	2025-10-31 17:58:12.466018	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
84	2025-10-31 17:58:12.480627	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
85	2025-10-31 17:58:20.452176	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
86	2025-10-31 17:58:21.429071	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
87	2025-10-31 17:58:22.466285	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
88	2025-10-31 17:58:22.474276	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
101	2025-10-31 17:59:00.465659	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
102	2025-10-31 17:59:01.432639	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
103	2025-10-31 17:59:02.471992	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
104	2025-10-31 17:59:02.475023	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
105	2025-10-31 17:59:10.472204	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
106	2025-10-31 17:59:11.43395	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
107	2025-10-31 17:59:12.475168	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
108	2025-10-31 17:59:12.50251	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
109	2025-10-31 17:59:20.478478	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
110	2025-10-31 17:59:21.435417	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
111	2025-10-31 17:59:22.478298	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
112	2025-10-31 17:59:22.482028	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
125	2025-10-31 18:00:00.506271	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
126	2025-10-31 18:00:01.444928	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
127	2025-10-31 18:00:02.520698	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
128	2025-10-31 18:00:02.523075	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
129	2025-10-31 18:00:10.511863	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
130	2025-10-31 18:00:11.445532	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
131	2025-10-31 18:00:12.524913	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
132	2025-10-31 18:00:12.562644	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
89	2025-10-31 17:58:30.454183	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
90	2025-10-31 17:58:31.4304	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
91	2025-10-31 17:58:32.466686	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
92	2025-10-31 17:58:32.511549	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
93	2025-10-31 17:58:40.458608	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
94	2025-10-31 17:58:41.431633	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
95	2025-10-31 17:58:42.466935	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
96	2025-10-31 17:58:42.471444	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
97	2025-10-31 17:58:50.465069	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
98	2025-10-31 17:58:51.432399	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
99	2025-10-31 17:58:52.46812	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
100	2025-10-31 17:58:52.472082	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
113	2025-10-31 17:59:30.48016	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
114	2025-10-31 17:59:31.435812	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
115	2025-10-31 17:59:32.500139	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
116	2025-10-31 17:59:32.510454	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
117	2025-10-31 17:59:40.494324	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
118	2025-10-31 17:59:41.437388	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
119	2025-10-31 17:59:42.506602	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
120	2025-10-31 17:59:42.745192	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
121	2025-10-31 17:59:50.50465	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
122	2025-10-31 17:59:51.43857	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
123	2025-10-31 17:59:52.506844	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
124	2025-10-31 17:59:52.510111	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
133	2025-10-31 18:00:20.529932	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
134	2025-10-31 18:00:21.447122	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
135	2025-10-31 18:00:22.528764	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
136	2025-10-31 18:00:22.55716	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
137	2025-10-31 18:00:30.530542	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
138	2025-10-31 18:00:31.448629	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
139	2025-10-31 18:00:32.527441	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
140	2025-10-31 18:00:32.534614	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
141	2025-10-31 18:00:40.531427	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
142	2025-10-31 18:00:41.449526	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
143	2025-10-31 18:00:42.529795	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
144	2025-10-31 18:00:42.550214	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
145	2025-10-31 18:00:50.538843	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
146	2025-10-31 18:00:51.45084	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
147	2025-10-31 18:00:52.529508	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_notify: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_notify"}
148	2025-10-31 18:00:52.532864	pgbridge	LISTENER_ERROR	Troubled Water	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
149	2025-10-31 18:01:00.544373	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: timeout: context deadline exceeded	{"error": "timeout: context deadline exceeded", "channel": "pgb_mail"}
150	2025-10-31 19:02:25.231012	pgbridge	SERVICE_START	River	\N	pgbridge service connected and initialized	{"version": "1.0.0", "initialized_at": "2025-10-31T19:02:25-04:00"}
151	2025-10-31 19:02:25.247412	pgbridge	MODULE_INIT	River	pgb_mail	Initializing module pgb_mail for database River	\N
152	2025-10-31 19:02:25.273584	pgbridge	MODULE_START	River	pgb_mail	Started module pgb_mail for database River	\N
153	2025-10-31 19:02:25.317868	pgbridge	LISTENER_STARTED	River	\N	Started listening on channel pgb_mail	{"channel": "pgb_mail"}
154	2025-10-31 19:02:25.32729	pgbridge	MODULE_INIT	River	pgb_notify	Initializing module pgb_notify for database River	\N
155	2025-10-31 19:02:25.497525	pgbridge	MODULE_START	River	pgb_notify	Started module pgb_notify for database River	\N
156	2025-10-31 19:02:25.511019	pgbridge	LISTENER_STARTED	River	\N	Started listening on channel pgb_notify	{"channel": "pgb_notify"}
157	2025-10-31 19:02:25.657439	pgbridge	MODULE_INIT	Troubled Water	pgb_mail	Initializing module pgb_mail for database Troubled Water	\N
158	2025-10-31 19:02:25.741592	pgbridge	MODULE_START	Troubled Water	pgb_mail	Started module pgb_mail for database Troubled Water	\N
159	2025-10-31 19:02:25.884336	pgbridge	LISTENER_STARTED	Troubled Water	\N	Started listening on channel pgb_mail	{"channel": "pgb_mail"}
160	2025-10-31 19:02:25.920737	pgbridge	MODULE_INIT	Troubled Water	pgb_notify	Initializing module pgb_notify for database Troubled Water	\N
161	2025-10-31 19:02:25.983189	pgbridge	MODULE_START	Troubled Water	pgb_notify	Started module pgb_notify for database Troubled Water	\N
162	2025-10-31 19:02:26.0365	pgbridge	LISTENER_STARTED	Troubled Water	\N	Started listening on channel pgb_notify	{"channel": "pgb_notify"}
163	2025-10-31 19:02:26.047789	pgbridge	SERVICE_START	\N	\N	pgbridge 1.0.0 started successfully with 2 databases	{"version": "1.0.0", "database_count": 2}
164	2025-10-31 19:06:20.143942	pgbridge	NOTIFY_FORWARDED	River	pgb_notify	Notification forwarded: source_id=1, central_id=1	{"source_id": 1, "central_id": 1}
165	2025-10-31 19:08:30.783917	pgbridge	LISTENER_ERROR	River	\N	Listener error on channel pgb_mail: read tcp [::1]:38140->[::1]:5432: use of closed network connection	{"error": "read tcp [::1]:38140->[::1]:5432: use of closed network connection", "channel": "pgb_mail"}
166	2025-11-02 06:55:37.007588	pgbridge	SERVICE_START	River river	\N	pgbridge service connected and initialized	{"version": "1.0.0", "initialized_at": "2025-11-02T06:55:37-05:00"}
167	2025-11-02 06:55:37.012166	pgbridge	MODULE_INIT	River river	pgb_notify	Initializing module pgb_notify for database River river	\N
168	2025-11-02 06:55:37.039656	pgbridge	MODULE_START	River river	pgb_notify	Started module pgb_notify for database River river	\N
169	2025-11-02 06:55:37.045614	pgbridge	LISTENER_STARTED	River river	\N	Started listening on channel pgb_notify	{"channel": "pgb_notify"}
170	2025-11-02 06:55:37.05048	pgbridge	MODULE_INIT	River river	pgb_mail	Initializing module pgb_mail for database River river	\N
171	2025-11-02 06:55:37.056848	pgbridge	MODULE_START	River river	pgb_mail	Started module pgb_mail for database River river	\N
172	2025-11-02 06:55:37.060502	pgbridge	LISTENER_STARTED	River river	\N	Started listening on channel pgb_mail	{"channel": "pgb_mail"}
173	2025-11-02 06:55:37.063194	pgbridge	SERVICE_START	\N	\N	pgbridge 1.0.0 started successfully with 1 databases	{"version": "1.0.0", "database_count": 1}
174	2025-11-02 06:57:44.316629	pgbridge	NOTIFY_FORWARDED	River river	pgb_notify	Notification forwarded: source_id=2, central_id=2	{"source_id": 2, "central_id": 2}
175	2025-11-02 07:14:34.035844	pgbridge	SERVICE_START	River river	\N	pgbridge service connected and initialized	{"version": "1.0.0", "initialized_at": "2025-11-02T07:14:34-05:00"}
176	2025-11-02 07:14:52.749031	pgbridge	LISTENER_ERROR	River river	\N	Listener error on channel pgb_notify: read tcp [::1]:56368->[::1]:5432: use of closed network connection	{"error": "read tcp [::1]:56368->[::1]:5432: use of closed network connection", "channel": "pgb_notify"}
177	2025-11-02 07:15:11.407562	pgbridge	SERVICE_START	River river	\N	pgbridge service connected and initialized	{"version": "1.0.0", "initialized_at": "2025-11-02T07:15:11-05:00"}
178	2025-11-02 10:27:49.032775	pgbridge	SERVICE_START	River river	\N	pgbridge service connected and initialized	{"version": "1.0.0", "initialized_at": "2025-11-02T10:27:49-05:00"}
179	2025-11-02 10:29:40.90043	pgbridge	SERVICE_START	River river	\N	pgbridge service connected and initialized	{"version": "1.0.0", "initialized_at": "2025-11-02T10:29:40-05:00"}
180	2025-11-02 10:30:00.689082	pgbridge	SERVICE_START	River river	\N	pgbridge service connected and initialized	{"version": "1.0.0", "initialized_at": "2025-11-02T10:30:00-05:00"}
\.


--
-- TOC entry 4292 (class 0 OID 19030)
-- Dependencies: 223
-- Data for Name: pgb_mail; Type: TABLE DATA; Schema: pgb; Owner: tyutyu
--

COPY pgb.pgb_mail (id, mail_setting_id, header_from, header_to, header_cc, header_bcc, subject, body_text, is_sent, sent_ts, error_message, retry_count, created_at, updated_at) FROM stdin;
\.


--
-- TOC entry 4290 (class 0 OID 19015)
-- Dependencies: 221
-- Data for Name: pgb_mail_settings; Type: TABLE DATA; Schema: pgb; Owner: tyutyu
--

COPY pgb.pgb_mail_settings (id, smtp_server, smtp_port, is_tls, is_ssl, smtp_user, smtp_password, smtp_token, created_at, updated_at) FROM stdin;
\.


--
-- TOC entry 4294 (class 0 OID 19053)
-- Dependencies: 225
-- Data for Name: pgb_notify; Type: TABLE DATA; Schema: pgb; Owner: tyutyu
--

COPY pgb.pgb_notify (id, user_email, sender_db, message, message_link, criticality, is_sent, sent_ts, is_seen, created_at, updated_at) FROM stdin;
1	botondzalai@pansoinco.com	river	This is our first distributed message	\N	2	t	2025-10-31 19:06:20.135607	f	2025-11-01 00:06:00.014126	2025-10-31 19:06:20.135607
2	botond@pansoinco.com	river	Testing second time	telex.hu	1	t	2025-11-02 06:57:44.3114	f	2025-11-02 12:57:44.294364	2025-11-02 06:57:44.3114
\.


--
-- TOC entry 4320 (class 0 OID 0)
-- Dependencies: 218
-- Name: pgb_log_id_seq; Type: SEQUENCE SET; Schema: pgb; Owner: tyutyu
--

SELECT pg_catalog.setval('pgb.pgb_log_id_seq', 180, true);


--
-- TOC entry 4321 (class 0 OID 0)
-- Dependencies: 222
-- Name: pgb_mail_id_seq; Type: SEQUENCE SET; Schema: pgb; Owner: tyutyu
--

SELECT pg_catalog.setval('pgb.pgb_mail_id_seq', 1, false);


--
-- TOC entry 4322 (class 0 OID 0)
-- Dependencies: 220
-- Name: pgb_mail_settings_id_seq; Type: SEQUENCE SET; Schema: pgb; Owner: tyutyu
--

SELECT pg_catalog.setval('pgb.pgb_mail_settings_id_seq', 1, false);


--
-- TOC entry 4323 (class 0 OID 0)
-- Dependencies: 224
-- Name: pgb_notify_id_seq; Type: SEQUENCE SET; Schema: pgb; Owner: tyutyu
--

SELECT pg_catalog.setval('pgb.pgb_notify_id_seq', 2, true);


--
-- TOC entry 4125 (class 2606 OID 19011)
-- Name: pgb_log pgb_log_pkey; Type: CONSTRAINT; Schema: pgb; Owner: tyutyu
--

ALTER TABLE ONLY pgb.pgb_log
    ADD CONSTRAINT pgb_log_pkey PRIMARY KEY (id);


--
-- TOC entry 4132 (class 2606 OID 19043)
-- Name: pgb_mail pgb_mail_pkey; Type: CONSTRAINT; Schema: pgb; Owner: tyutyu
--

ALTER TABLE ONLY pgb.pgb_mail
    ADD CONSTRAINT pgb_mail_pkey PRIMARY KEY (id);


--
-- TOC entry 4127 (class 2606 OID 19028)
-- Name: pgb_mail_settings pgb_mail_settings_pkey; Type: CONSTRAINT; Schema: pgb; Owner: tyutyu
--

ALTER TABLE ONLY pgb.pgb_mail_settings
    ADD CONSTRAINT pgb_mail_settings_pkey PRIMARY KEY (id);


--
-- TOC entry 4138 (class 2606 OID 19066)
-- Name: pgb_notify pgb_notify_pkey; Type: CONSTRAINT; Schema: pgb; Owner: tyutyu
--

ALTER TABLE ONLY pgb.pgb_notify
    ADD CONSTRAINT pgb_notify_pkey PRIMARY KEY (id);


--
-- TOC entry 4122 (class 1259 OID 19013)
-- Name: idx_pgb_log_event_type; Type: INDEX; Schema: pgb; Owner: tyutyu
--

CREATE INDEX idx_pgb_log_event_type ON pgb.pgb_log USING btree (event_type);


--
-- TOC entry 4123 (class 1259 OID 19012)
-- Name: idx_pgb_log_timestamp; Type: INDEX; Schema: pgb; Owner: tyutyu
--

CREATE INDEX idx_pgb_log_timestamp ON pgb.pgb_log USING btree ("timestamp");


--
-- TOC entry 4128 (class 1259 OID 19050)
-- Name: idx_pgb_mail_created_at; Type: INDEX; Schema: pgb; Owner: tyutyu
--

CREATE INDEX idx_pgb_mail_created_at ON pgb.pgb_mail USING btree (created_at);


--
-- TOC entry 4129 (class 1259 OID 19049)
-- Name: idx_pgb_mail_is_sent; Type: INDEX; Schema: pgb; Owner: tyutyu
--

CREATE INDEX idx_pgb_mail_is_sent ON pgb.pgb_mail USING btree (is_sent) WHERE (is_sent = false);


--
-- TOC entry 4130 (class 1259 OID 19051)
-- Name: idx_pgb_mail_setting_id; Type: INDEX; Schema: pgb; Owner: tyutyu
--

CREATE INDEX idx_pgb_mail_setting_id ON pgb.pgb_mail USING btree (mail_setting_id);


--
-- TOC entry 4133 (class 1259 OID 19069)
-- Name: idx_pgb_notify_created_at; Type: INDEX; Schema: pgb; Owner: tyutyu
--

CREATE INDEX idx_pgb_notify_created_at ON pgb.pgb_notify USING btree (created_at);


--
-- TOC entry 4134 (class 1259 OID 19070)
-- Name: idx_pgb_notify_is_seen; Type: INDEX; Schema: pgb; Owner: tyutyu
--

CREATE INDEX idx_pgb_notify_is_seen ON pgb.pgb_notify USING btree (is_seen) WHERE (is_seen = false);


--
-- TOC entry 4135 (class 1259 OID 19067)
-- Name: idx_pgb_notify_is_sent; Type: INDEX; Schema: pgb; Owner: tyutyu
--

CREATE INDEX idx_pgb_notify_is_sent ON pgb.pgb_notify USING btree (is_sent) WHERE (is_sent = false);


--
-- TOC entry 4136 (class 1259 OID 19068)
-- Name: idx_pgb_notify_user_email; Type: INDEX; Schema: pgb; Owner: tyutyu
--

CREATE INDEX idx_pgb_notify_user_email ON pgb.pgb_notify USING btree (user_email);


--
-- TOC entry 4140 (class 2620 OID 19354)
-- Name: pgb_notify S01_send_notification; Type: TRIGGER; Schema: pgb; Owner: tyutyu
--

CREATE TRIGGER "S01_send_notification" BEFORE INSERT ON pgb.pgb_notify FOR EACH ROW EXECUTE FUNCTION pgb.trg_pgb_send_notification();


--
-- TOC entry 4141 (class 2620 OID 19383)
-- Name: pgb_notify s01_send_notification; Type: TRIGGER; Schema: pgb; Owner: tyutyu
--

CREATE TRIGGER s01_send_notification AFTER INSERT ON pgb.pgb_notify FOR EACH ROW EXECUTE FUNCTION pgb.trg_pgb_send_notification();


--
-- TOC entry 4324 (class 0 OID 0)
-- Dependencies: 4141
-- Name: TRIGGER s01_send_notification ON pgb_notify; Type: COMMENT; Schema: pgb; Owner: tyutyu
--

COMMENT ON TRIGGER s01_send_notification ON pgb.pgb_notify IS 'Automatically triggers notification forwarding on insert';


--
-- TOC entry 4139 (class 2606 OID 19044)
-- Name: pgb_mail fk_mail_setting; Type: FK CONSTRAINT; Schema: pgb; Owner: tyutyu
--

ALTER TABLE ONLY pgb.pgb_mail
    ADD CONSTRAINT fk_mail_setting FOREIGN KEY (mail_setting_id) REFERENCES pgb.pgb_mail_settings(id) ON DELETE RESTRICT;


--
-- TOC entry 4301 (class 0 OID 0)
-- Dependencies: 6
-- Name: SCHEMA public; Type: ACL; Schema: -; Owner: pg_database_owner
--

GRANT USAGE ON SCHEMA public TO riverside;


-- Completed on 2025-11-02 11:19:30 EST

--
-- PostgreSQL database dump complete
--

-- Completed on 2025-11-02 11:19:30 EST

--
-- PostgreSQL database cluster dump complete
--

