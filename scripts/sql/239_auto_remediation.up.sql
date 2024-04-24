CREATE SEQUENCE IF NOT EXISTS id_seq_watcher;
CREATE TABLE "public"."watcher" (
                                    "id" integer NOT NULL default nextval('id_seq_watcher'::regclass),
                                    "name" varchar(100) NOT NULL,
                                    "desc" text ,
                                    "filter_expression" text NOT NULL,
                                    "gvks" text[],
                                    "active" bool,

                                    PRIMARY KEY ("id"),
                                        UNIQUE ("name")

);
CREATE UNIQUE INDEX "idx_unique_watcher_name"
    ON watcher(name)
    WHERE (active=true);

CREATE SEQUENCE IF NOT EXISTS id_seq_trigger;
CREATE TABLE "public"."trigger"(
                                   "id" integer NOT NULL default nextval('id_seq_trigger'::regclass),
                                   "type" varchar(255) , -- DEVTRON_JOB
                                   "watcher_id" integer ,
                                   "data" JSON,

                                   CONSTRAINT trigger_watcher_id_fkey FOREIGN KEY ("watcher_id") REFERENCES "public"."watcher" ("id"),
                                   PRIMARY KEY ("id")

);

CREATE SEQUENCE IF NOT EXISTS id_seq_intercepted_events;
CREATE TABLE "public"."intercepted_events"(
                                              "id" integer NOT NULL default nextval('id_seq_intercepted_events'::regclass),
                                              "cluster_id" integer ,
                                              "namespace" varchar(100),
                                              "event" text,
                                              "involved_object" text,
                                              "intercepted_at" timestamptz,
                                              "status" varchar(255) , -- failure,success,inprogress
                                              "trigger_id" integer,
                                              "trigger_execution_id" integer,

                                              CONSTRAINT intercepted_events_trigger_id_fkey FOREIGN KEY ("trigger_id") REFERENCES "public"."trigger" ("id"),
                                              PRIMARY KEY ("id")
);

