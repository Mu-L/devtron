CREATE SEQUENCE IF NOT EXISTS id_seq_image_tag;

CREATE TABLE IF NOT EXISTS public.release_tags (
    "id"                                         integer NOT NULL DEFAULT nextval('id_seq_image_tag'::regclass),
    "tag_name"                                   varchar(128),
    "artifact_id"                                integer,
    "active"                                        BOOL,
    "app_id"                                     integer,
    CONSTRAINT "image_tag_app_id_fkey" FOREIGN KEY ("app_id") REFERENCES "public"."app" ("id"),
    CONSTRAINT "image_tag_artifact_id_fkey" FOREIGN KEY ("artifact_id") REFERENCES "public"."ci_artifact" ("id"),
    UNIQUE ("app_id","tag_name"),
    PRIMARY KEY ("id")
    );

CREATE SEQUENCE IF NOT EXISTS id_seq_image_comment;

CREATE TABLE IF NOT EXISTS public.image_comments (
    "id"                                         integer NOT NULL DEFAULT nextval('id_seq_image_comment'::regclass),
    "comment"                                    text,
    "artifact_id"                                integer,
    "user_id"                                    integer,
    CONSTRAINT "image_comment_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id"),
    CONSTRAINT "image_comment_artifact_id_fkey" FOREIGN KEY ("artifact_id") REFERENCES "public"."ci_artifact" ("id"),
    PRIMARY KEY ("id")
    );

CREATE SEQUENCE IF NOT EXISTS id_seq_image_tagging_audit;
CREATE TYPE IF NOT EXISTS image_tagging_data_type AS ENUM ('TAG','COMMENT');
CREATE TYPE IF NOT EXISTS action_type AS ENUM ('SAVE','EDIT','SOFT_DELETE','HARD_DELETE');

CREATE TABLE IF NOT EXISTS public.image_tagging_audit (
    "id"                                      integer NOT NULL DEFAULT nextval('id_seq_image_tagging_audit'::regclass),
    "data"                                    text,
    "data_type"                               image_tagging_data_type,
    "artifact_id"                             integer,
    "action"                                  action_type,
    "updated_on"                              timestamptz,
    "updated_by"                              integer,
    CONSTRAINT "image_tagging_audit_updated_by_fkey" FOREIGN KEY ("updated_by") REFERENCES "public"."users" ("id"),
    CONSTRAINT "image_tagging_audit_artifact_id_fkey" FOREIGN KEY ("artifact_id") REFERENCES "public"."ci_artifact" ("id"),
    PRIMARY KEY ("id")
    );