ALTER TABLE payment_orders
    ALTER COLUMN amount TYPE DECIMAL(20,8) USING amount::DECIMAL(20,8),
    ALTER COLUMN pay_amount TYPE DECIMAL(20,8) USING pay_amount::DECIMAL(20,8),
    ALTER COLUMN refund_amount TYPE DECIMAL(20,8) USING refund_amount::DECIMAL(20,8);
