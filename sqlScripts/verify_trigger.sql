CREATE OR REPLACE FUNCTION notify_verification()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.is_verified = TRUE AND OLD.is_verified = FALSE THEN
        PERFORM pg_notify('verify_channel', json_build_object(
        'username', NEW.username,
        'mail', NEW.mail
        )::text);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER check_verification
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION notify_verification();