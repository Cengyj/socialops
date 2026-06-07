-- Align redeem_codes.code with generated 35-character codes and the existing
-- admin fixed-code endpoint contract (max 128 characters).
ALTER TABLE redeem_codes
    ALTER COLUMN code TYPE VARCHAR(128);
