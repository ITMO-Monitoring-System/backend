-- Добавление ON DELETE CASCADE для FK, привязанных к жизни студента.
-- Удаление пользователя теперь автоматически чистит зависимости в:
--   cores.face_images, cores.users_passwords, cores.users_roles,
--   universities_data.students_groups,
--   visits.lectures_visiting, visits.practices_visiting.
-- FK universities_data.lectures.teacher_id / practices.teacher_id оставлены
-- без CASCADE намеренно — удаление преподавателя должно явно фейлиться, чтобы
-- администратор сначала переназначил/удалил его лекции.

ALTER TABLE cores.face_images
    DROP CONSTRAINT face_images_student_id_fkey,
    ADD CONSTRAINT face_images_student_id_fkey
        FOREIGN KEY (student_id) REFERENCES cores.users(isu) ON DELETE CASCADE;

ALTER TABLE cores.users_passwords
    DROP CONSTRAINT users_passwords_isu_fkey,
    ADD CONSTRAINT users_passwords_isu_fkey
        FOREIGN KEY (isu) REFERENCES cores.users(isu) ON DELETE CASCADE;

ALTER TABLE cores.users_roles
    DROP CONSTRAINT users_roles_isu_fkey,
    ADD CONSTRAINT users_roles_isu_fkey
        FOREIGN KEY (isu) REFERENCES cores.users(isu) ON DELETE CASCADE;

ALTER TABLE universities_data.students_groups
    DROP CONSTRAINT students_groups_user_id_fkey,
    ADD CONSTRAINT students_groups_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES cores.users(isu) ON DELETE CASCADE;

ALTER TABLE visits.lectures_visiting
    DROP CONSTRAINT lectures_visiting_user_id_fkey,
    ADD CONSTRAINT lectures_visiting_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES cores.users(isu) ON DELETE CASCADE;

ALTER TABLE visits.practices_visiting
    DROP CONSTRAINT practices_visiting_user_id_fkey,
    ADD CONSTRAINT practices_visiting_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES cores.users(isu) ON DELETE CASCADE;
