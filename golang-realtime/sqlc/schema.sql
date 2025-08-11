-- WARNING: This schema is for context only and is not meant to be run.
-- Table order and constraints may not be valid for execution.

CREATE TABLE public.players (
  id integer NOT NULL DEFAULT nextval('players_id_seq'::regclass),
  name character varying NOT NULL UNIQUE,
  password text NOT NULL,
  CONSTRAINT players_pkey PRIMARY KEY (id)
);
CREATE TABLE public.programming_languages (
  id integer NOT NULL DEFAULT nextval('programming_languages_id_seq'::regclass),
  name character varying NOT NULL UNIQUE,
  CONSTRAINT programming_languages_pkey PRIMARY KEY (id)
);
CREATE TABLE public.questions (
  id integer NOT NULL DEFAULT nextval('questions_id_seq'::regclass),
  language_id integer NOT NULL,
  template_function text,
  title character varying NOT NULL,
  description text,
  score integer NOT NULL,
  difficulty integer NOT NULL,
  CONSTRAINT questions_pkey PRIMARY KEY (id, language_id),
  CONSTRAINT questions_language_id_fkey FOREIGN KEY (language_id) REFERENCES public.programming_languages(id)
);
CREATE TABLE public.room_players (
  room_id integer NOT NULL,
  player_id integer NOT NULL,
  score integer DEFAULT 0,
  place integer,
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
CREATE TABLE public.test_cases (
  id integer NOT NULL DEFAULT nextval('test_cases_id_seq'::regclass),
  question_id integer NOT NULL,
  question_language_id integer NOT NULL,
  input text NOT NULL,
  expected_output text NOT NULL,
  time_constraint double precision,
  space_constraint integer,
  CONSTRAINT test_cases_pkey PRIMARY KEY (id),
  CONSTRAINT test_cases_question_id_question_language_id_fkey FOREIGN KEY (question_language_id) REFERENCES public.questions(id),
  CONSTRAINT test_cases_question_id_question_language_id_fkey FOREIGN KEY (question_id) REFERENCES public.questions(id),
  CONSTRAINT test_cases_question_id_question_language_id_fkey FOREIGN KEY (question_language_id) REFERENCES public.questions(language_id),
  CONSTRAINT test_cases_question_id_question_language_id_fkey FOREIGN KEY (question_id) REFERENCES public.questions(language_id)
);
