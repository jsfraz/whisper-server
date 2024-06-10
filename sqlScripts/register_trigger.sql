CREATE OR REPLACE FUNCTION register_trigger()
RETURNS TRIGGER AS $$
BEGIN
    PERFORM pg_notify('register_channel', json_build_object(
        'username', NEW.username,
        'mail', NEW.mail,
        'verification_code', NEW.verification_code
    )::text);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER register_trigger
AFTER INSERT ON users
FOR EACH ROW
EXECUTE FUNCTION register_trigger();