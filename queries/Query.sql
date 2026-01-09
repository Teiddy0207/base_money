SELECT
    id,
    email,
    username,
    LENGTH(username) as username_length,
    TRIM(username) as username_trimmed,
    LENGTH(TRIM(username)) as trimmed_length,
    encode(username::bytea, 'hex') as username_hex,
    encode(TRIM(username)::bytea, 'hex') as trimmed_hex
FROM social_logins
WHERE username ILIKE '%Kiên%' OR username ILIKE '%Nguyễn%'
ORDER BY created_at DESC;