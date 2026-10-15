-- This rollback requires removing duplicate names and aliases first.
ALTER TABLE universities_data.departments
    ADD CONSTRAINT departments_name_key UNIQUE (name),
    ADD CONSTRAINT departments_alias_key UNIQUE (alias);
