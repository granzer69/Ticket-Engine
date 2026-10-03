-- Post-load inventory reconciliation (run after k6 test.js).
SELECT COUNT(*) AS total_rows FROM tickets;
SELECT COUNT(*) AS sold_rows FROM tickets WHERE state = 'sold';
SELECT COUNT(DISTINCT user_id) AS distinct_buyers FROM tickets WHERE state = 'sold' AND user_id IS NOT NULL;
SELECT user_id, COUNT(*) AS c FROM tickets WHERE state = 'sold' AND user_id IS NOT NULL GROUP BY user_id HAVING c > 1;
SELECT id, COUNT(*) AS c FROM tickets WHERE state = 'sold' GROUP BY id HAVING c > 1;
