DROP TABLE IF EXISTS image_tags;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS images;
DROP TABLE IF EXISTS album_images;
DROP TABLE IF EXISTS post_images;

DROP INDEX IF EXISTS images_profile_id_idx; 
DROP INDEX IF EXISTS images_image_id_idx;
DROP INDEX IF EXISTS images_created_at_idx;
DROP INDEX IF EXISTS images_score_idx;

DROP INDEX IF EXISTS tags_name_idx;
DROP INDEX IF EXISTS tags_name_unique;

DROP INDEX IF EXISTS image_tags_image_id_idx;
DROP INDEX IF EXISTS image_tags_tag_id_idx;

DROP INDEX IF EXISTS album_images_album_id_idx;
DROP INDEX IF EXISTS album_images_image_id_idx;

DROP INDEX IF EXISTS post_images_post_id_idx;
DROP INDEX IF EXISTS post_images_image_id_idx;