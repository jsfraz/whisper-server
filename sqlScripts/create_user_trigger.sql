CREATE OR REPLACE FUNCTION create_user_trigger()
RETURNS TRIGGER AS $$
BEGIN
    PERFORM pg_notify('create_user_channel', NEW.id::text);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER create_user_trigger
AFTER INSERT ON users
FOR EACH ROW
EXECUTE FUNCTION create_user_trigger();