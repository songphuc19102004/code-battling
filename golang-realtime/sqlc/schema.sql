-- WARNING: This schema is for context only and is not meant to be run.
-- Table order and constraints may not be valid for execution.

CREATE TABLE public.languages (
  id integer NOT NULL DEFAULT nextval('languages_id_seq'::regclass),
  name character varying NOT NULL UNIQUE,
  compile_cmd character varying,
  run_cmd character varying,
  timeout_second double precision,
  CONSTRAINT languages_pkey PRIMARY KEY (id)
);
CREATE TABLE public.players (
  id integer NOT NULL DEFAULT nextval('players_id_seq'::regclass),
  name character varying NOT NULL UNIQUE,
  password text NOT NULL,
  CONSTRAINT players_pkey PRIMARY KEY (id)
);
CREATE TABLE public.questions (
  id integer NOT NULL,
  language_id integer NOT NULL,
  template_function text,
  title character varying NOT NULL,
  description text,
  score integer NOT NULL,
  difficulty integer NOT NULL,
  CONSTRAINT questions_pkey PRIMARY KEY (id, language_id),
  CONSTRAINT questions_language_id_fkey FOREIGN KEY (language_id) REFERENCES public.languages(id)
);
CREATE TABLE public.room_players (
  room_id integer NOT NULL,
  player_id integer NOT NULL,
  score integer DEFAULT 0,
  place integer,
  state text DEFAULT NULL,
  CONSTRAINT room_players_pkey PRIMARY KEY (room_id, player_id),
  CONSTRAINT room_players_player_id_fkey FOREIGN KEY (player_id) REFERENCES public.players(id),
  CONSTRAINT room_players_room_id_fkey FOREIGN KEY (room_id) REFERENCES public.rooms(id)
);
CREATE TABLE public.rooms (
  id integer NOT NULL DEFAULT nextval('rooms_id_seq'::regclass),
  name character varying NOT NULL,
  description text,
  CONSTRAINT rooms_pkey PRIMARY KEY (id)
);
CREATE TABLE public.submissions (
  id integer NOT NULL DEFAULT nextval('submissions_id_seq'::regclass),
  source_code text,
  language_id integer,
  stdin text,
  expected_output text,
  stdout text,
  status_id integer,
  created_at timestamp without time zone,
  finished_at timestamp without time zone,
  time numeric,
  memory integer,
  stderr text,
  token character varying,
  number_of_runs integer,
  cpu_time_limit numeric,
  cpu_extra_time numeric,
  wall_time_limit numeric,
  memory_limit integer,
  stack_limit integer,
  max_processes_and_or_threads integer,
  enable_per_process_and_thread_time_limit boolean,
  enable_per_process_and_thread_memory_limit boolean,
  max_file_size integer,
  compile_output text,
  exit_code integer,
  exit_signal integer,
  message text,
  wall_time numeric,
  compiler_options character varying,
  command_line_arguments character varying,
  redirect_stderr_to_stdout boolean,
  callback_url character varying,
  additional_files bytea,
  enable_network boolean,
  started_at timestamp without time zone,
  queued_at timestamp without time zone,
  updated_at timestamp without time zone,
  queue_host character varying,
  execution_host character varying,
  CONSTRAINT submissions_pkey PRIMARY KEY (id),
  CONSTRAINT fk_languages FOREIGN KEY (language_id) REFERENCES public.languages(id)
);
CREATE TABLE public.test_cases (
  id integer NOT NULL DEFAULT nextval('test_cases_id_seq'::regclass),
  question_id integer NOT NULL,
  question_language_id integer NOT NULL,
  input text NOT NULL,
  expected_output text NOT NULL,
  time_constraint double precision,
  space_constraint integer,
  CONSTRAINT test_cases_pkey PRIMARY KEY (id)
);
