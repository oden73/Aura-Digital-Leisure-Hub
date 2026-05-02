-- Seed data: 2 users, 30 catalog items (books/cinema/games), interactions.
-- Passwords are bcrypt-hashed via pgcrypto: both users have password "seed1234".
-- d stands for games (g is invalid for a seed)

BEGIN;

-- ── Users ─────────────────────────────────────────────────────────────────

INSERT INTO users (user_id, username, email, password_hash) VALUES
    ('a1000000-0000-0000-0000-000000000001', 'alice', 'alice@example.com',
     crypt('seed1234', gen_salt('bf', 10))),
    ('a1000000-0000-0000-0000-000000000002', 'bob',   'bob@example.com',
     crypt('seed1234', gen_salt('bf', 10)));

-- ── Base items ────────────────────────────────────────────────────────────

INSERT INTO base_items
    (item_id, title, original_title, description, release_date, average_rating,
     media_type, genre, setting, themes, tonality, target_audience)
VALUES
-- Books (10)
('b0000000-0000-0000-0000-000000000001','Dune','Dune',
 'Epic sci-fi saga of desert planet Arrakis and the Atreides family.',
 '1965-08-01',9.10,'book','sci-fi','space/desert','power, ecology, religion','epic','adult'),
('b0000000-0000-0000-0000-000000000002','1984','Nineteen Eighty-Four',
 'Dystopian novel of totalitarian surveillance society.',
 '1949-06-08',9.20,'book','dystopia','urban','surveillance, freedom','dark','adult'),
('b0000000-0000-0000-0000-000000000003','The Name of the Wind','The Name of the Wind',
 'Coming-of-age story of legendary wizard Kvothe.',
 '2007-03-27',8.80,'book','fantasy','medieval','magic, identity','adventurous','young adult'),
('b0000000-0000-0000-0000-000000000004','Neuromancer','Neuromancer',
 'Seminal cyberpunk novel about a washed-up hacker hired for one last run.',
 '1984-07-01',8.50,'book','cyberpunk','dystopian city','technology, identity','dark','adult'),
('b0000000-0000-0000-0000-000000000005','The Hitchhiker''s Guide to the Galaxy',
 'The Hitchhiker''s Guide to the Galaxy',
 'Comic sci-fi following Arthur Dent across the galaxy.',
 '1979-10-12',9.00,'book','sci-fi','space','absurdism, adventure','humorous','adult'),
('b0000000-0000-0000-0000-000000000006','Sapiens','Sapiens: A Brief History of Humankind',
 'Non-fiction overview of Homo sapiens from Stone Age to present.',
 '2011-01-01',8.40,'book','non-fiction','global','history, society','informative','adult'),
('b0000000-0000-0000-0000-000000000007','The Left Hand of Darkness',
 'The Left Hand of Darkness',
 'A human envoy navigates gender-fluid alien society.',
 '1969-03-01',8.60,'book','sci-fi','alien world','gender, politics','thoughtful','adult'),
('b0000000-0000-0000-0000-000000000008','Good Omens','Good Omens',
 'An angel and a demon team up to prevent the apocalypse.',
 '1990-05-01',8.70,'book','fantasy','modern England','good vs evil','humorous','adult'),
('b0000000-0000-0000-0000-000000000009','Flowers for Algernon','Flowers for Algernon',
 'A mentally disabled man undergoes surgery to increase his intelligence.',
 '1966-03-01',8.90,'book','sci-fi','urban','intelligence, humanity','melancholic','adult'),
('b0000000-0000-0000-0000-000000000010','Snow Crash','Snow Crash',
 'A hacker and pizza deliveryman unravels a virtual-reality conspiracy.',
 '1992-06-01',8.30,'book','cyberpunk','near-future USA','technology, language','fast-paced','adult'),

-- Cinema (10)
('c0000000-0000-0000-0000-000000000001','Inception','Inception',
 'A thief enters dreams to plant an idea in a CEO''s mind.',
 '2010-07-16',8.80,'cinema','sci-fi/thriller','dreamscapes','reality, time','mind-bending','adult'),
('c0000000-0000-0000-0000-000000000002','Interstellar','Interstellar',
 'Astronauts travel through a wormhole to find a new home for humanity.',
 '2014-11-07',8.60,'cinema','sci-fi','space','love, time, survival','epic','adult'),
('c0000000-0000-0000-0000-000000000003','The Matrix','The Matrix',
 'A hacker learns his world is a simulation controlled by machines.',
 '1999-03-31',8.70,'cinema','sci-fi/action','dystopian simulation','reality, freedom','action-packed','adult'),
('c0000000-0000-0000-0000-000000000004','Parasite','Parasite',
 'A poor Korean family schemes to become employed by a wealthy family.',
 '2019-05-30',8.50,'cinema','thriller/drama','Seoul','class, deception','dark','adult'),
('c0000000-0000-0000-0000-000000000005','Spirited Away','Sen to Chihiro no kamikakushi',
 'A girl enters the spirit world to rescue her parents.',
 '2001-07-20',9.30,'cinema','animation/fantasy','spirit realm','identity, courage','whimsical','all ages'),
('c0000000-0000-0000-0000-000000000006','The Shawshank Redemption',
 'The Shawshank Redemption',
 'A banker copes with prison life after being wrongly convicted of murder.',
 '1994-09-23',9.30,'cinema','drama','prison','hope, friendship','emotional','adult'),
('c0000000-0000-0000-0000-000000000007','Blade Runner 2049','Blade Runner 2049',
 'A blade runner uncovers a secret that threatens civilization.',
 '2017-10-06',8.00,'cinema','sci-fi/noir','dystopian LA','identity, humanity','atmospheric','adult'),
('c0000000-0000-0000-0000-000000000008','Everything Everywhere All at Once',
 'Everything Everywhere All at Once',
 'A laundromat owner must connect with parallel versions of herself.',
 '2022-03-25',8.00,'cinema','sci-fi/action','multiverse','family, identity','chaotic-funny','adult'),
('c0000000-0000-0000-0000-000000000009','Arrival','Arrival',
 'A linguist deciphers alien language as the world prepares for war.',
 '2016-11-11',7.90,'cinema','sci-fi','Earth/alien ships','language, time, grief','contemplative','adult'),
('c0000000-0000-0000-0000-000000000010','The Grand Budapest Hotel',
 'The Grand Budapest Hotel',
 'A legendary concierge and his protégé become embroiled in a crime caper.',
 '2014-02-26',8.10,'cinema','comedy/adventure','fictional Europe','friendship, nostalgia','quirky','adult'),

-- Games (10)
('d0000000-0000-0000-0000-000000000001','The Witcher 3: Wild Hunt',
 'The Witcher 3: Wild Hunt',
 'An open-world RPG following monster hunter Geralt of Rivia.',
 '2015-05-19',9.60,'game','RPG','fantasy','choice, war, family','dark epic','adult'),
('d0000000-0000-0000-0000-000000000002','Red Dead Redemption 2',
 'Red Dead Redemption 2',
 'A tale of loyalty and outlaws in the dying American frontier.',
 '2018-10-26',9.70,'game','action-adventure','wild west','loyalty, freedom, death','melancholic','adult'),
('d0000000-0000-0000-0000-000000000003','Hades','Hades',
 'A prince of the Underworld fights his way to the surface repeatedly.',
 '2020-09-17',9.30,'game','roguelike/action','greek mythology','perseverance, family','fast-paced','teen+'),
('d0000000-0000-0000-0000-000000000004','Disco Elysium','Disco Elysium',
 'A detective with amnesia investigates a murder in a decaying city.',
 '2019-10-15',9.50,'game','RPG','noir city','identity, politics','dark comedic','adult'),
('d0000000-0000-0000-0000-000000000005','Portal 2','Portal 2',
 'A puzzle platformer using a portal gun to solve physics puzzles.',
 '2011-04-19',9.50,'game','puzzle','lab/space','science, humor','witty','all ages'),
('d0000000-0000-0000-0000-000000000006','Hollow Knight','Hollow Knight',
 'A knight explores a vast underground insect kingdom.',
 '2017-02-24',9.10,'game','metroidvania','underground','solitude, mystery','melancholic','teen+'),
('d0000000-0000-0000-0000-000000000007','Celeste','Celeste',
 'A young woman climbs a treacherous mountain while battling mental health.',
 '2018-01-25',9.30,'game','platformer','mountain','mental health, determination','emotional','all ages'),
('d0000000-0000-0000-0000-000000000008','Stardew Valley','Stardew Valley',
 'A city worker inherits a farm and builds a new life in a rural town.',
 '2016-02-26',9.20,'game','simulation','rural farm','community, nature','cozy','all ages'),
('d0000000-0000-0000-0000-000000000009','Elden Ring','Elden Ring',
 'A soulslike set in a vast open world crafted by Hidetaka Miyazaki and George R.R. Martin.',
 '2022-02-25',9.50,'game','action RPG','dark fantasy','will, death, cycles','brutal epic','adult'),
('d0000000-0000-0000-0000-000000000010','Outer Wilds','Outer Wilds',
 'An astronaut explores a solar system caught in a 22-minute time loop.',
 '2019-05-28',9.60,'game','exploration/puzzle','space','curiosity, mortality','wonder','adult');

-- ── Media-specific details ─────────────────────────────────────────────────

INSERT INTO book_details (item_id, author, publisher, page_count) VALUES
('b0000000-0000-0000-0000-000000000001','Frank Herbert','Chilton Books',412),
('b0000000-0000-0000-0000-000000000002','George Orwell','Secker & Warburg',328),
('b0000000-0000-0000-0000-000000000003','Patrick Rothfuss','DAW Books',662),
('b0000000-0000-0000-0000-000000000004','William Gibson','Ace Books',271),
('b0000000-0000-0000-0000-000000000005','Douglas Adams','Pan Books',224),
('b0000000-0000-0000-0000-000000000006','Yuval Noah Harari','Dvir Publishing',443),
('b0000000-0000-0000-0000-000000000007','Ursula K. Le Guin','Ace Books',286),
('b0000000-0000-0000-0000-000000000008','Terry Pratchett, Neil Gaiman','Gollancz',288),
('b0000000-0000-0000-0000-000000000009','Daniel Keyes','Harcourt',311),
('b0000000-0000-0000-0000-000000000010','Neal Stephenson','Bantam Books',440);

INSERT INTO cinema_details (item_id, director, duration_mins) VALUES
('c0000000-0000-0000-0000-000000000001','Christopher Nolan',148),
('c0000000-0000-0000-0000-000000000002','Christopher Nolan',169),
('c0000000-0000-0000-0000-000000000003','Lana Wachowski, Lilly Wachowski',136),
('c0000000-0000-0000-0000-000000000004','Bong Joon-ho',132),
('c0000000-0000-0000-0000-000000000005','Hayao Miyazaki',125),
('c0000000-0000-0000-0000-000000000006','Frank Darabont',142),
('c0000000-0000-0000-0000-000000000007','Denis Villeneuve',164),
('c0000000-0000-0000-0000-000000000008','Daniel Kwan, Daniel Scheinert',139),
('c0000000-0000-0000-0000-000000000009','Denis Villeneuve',116),
('c0000000-0000-0000-0000-000000000010','Wes Anderson',99);

INSERT INTO game_details (item_id, developer, gameplay_genre, platforms, player_count) VALUES
('d0000000-0000-0000-0000-000000000001','CD Projekt Red','RPG','PC, PS4, PS5, Xbox','single-player'),
('d0000000-0000-0000-0000-000000000002','Rockstar Games','action-adventure','PC, PS4, PS5, Xbox','single-player'),
('d0000000-0000-0000-0000-000000000003','Supergiant Games','roguelike','PC, Switch, PS4','single-player'),
('d0000000-0000-0000-0000-000000000004','ZA/UM','RPG','PC, PS4, Xbox','single-player'),
('d0000000-0000-0000-0000-000000000005','Valve','puzzle platformer','PC, PS3, Xbox 360','single/co-op'),
('d0000000-0000-0000-0000-000000000006','Team Cherry','metroidvania','PC, Switch, PS4','single-player'),
('d0000000-0000-0000-0000-000000000007','Maddy Thorson','platformer','PC, Switch, PS4','single-player'),
('d0000000-0000-0000-0000-000000000008','ConcernedApe','simulation','PC, Switch, PS4','single/co-op'),
('d0000000-0000-0000-0000-000000000009','FromSoftware','action RPG','PC, PS4, PS5, Xbox','single/co-op'),
('d0000000-0000-0000-0000-000000000010','Mobius Digital','exploration','PC, PS4, Xbox','single-player');

-- ── Interactions ──────────────────────────────────────────────────────────

INSERT INTO user_interactions
    (user_id, item_id, status, rating, is_favorite)
VALUES
-- alice
('a1000000-0000-0000-0000-000000000001','b0000000-0000-0000-0000-000000000001','completed',9,true),
('a1000000-0000-0000-0000-000000000001','b0000000-0000-0000-0000-000000000002','completed',10,true),
('a1000000-0000-0000-0000-000000000001','b0000000-0000-0000-0000-000000000005','completed',8,false),
('a1000000-0000-0000-0000-000000000001','c0000000-0000-0000-0000-000000000001','completed',9,true),
('a1000000-0000-0000-0000-000000000001','c0000000-0000-0000-0000-000000000003','completed',10,true),
('a1000000-0000-0000-0000-000000000001','c0000000-0000-0000-0000-000000000009','completed',8,false),
('a1000000-0000-0000-0000-000000000001','d0000000-0000-0000-0000-000000000004','completed',10,true),
('a1000000-0000-0000-0000-000000000001','d0000000-0000-0000-0000-000000000005','completed',9,true),
('a1000000-0000-0000-0000-000000000001','d0000000-0000-0000-0000-000000000010','in_progress',NULL,false),
('a1000000-0000-0000-0000-000000000001','b0000000-0000-0000-0000-000000000004','planned',NULL,false),
('a1000000-0000-0000-0000-000000000001','c0000000-0000-0000-0000-000000000007','planned',NULL,false),
-- bob
('a1000000-0000-0000-0000-000000000002','d0000000-0000-0000-0000-000000000001','completed',10,true),
('a1000000-0000-0000-0000-000000000002','d0000000-0000-0000-0000-000000000002','completed',9,true),
('a1000000-0000-0000-0000-000000000002','d0000000-0000-0000-0000-000000000003','completed',9,false),
('a1000000-0000-0000-0000-000000000002','d0000000-0000-0000-0000-000000000006','completed',8,true),
('a1000000-0000-0000-0000-000000000002','d0000000-0000-0000-0000-000000000009','completed',10,true),
('a1000000-0000-0000-0000-000000000002','c0000000-0000-0000-0000-000000000004','completed',9,false),
('a1000000-0000-0000-0000-000000000002','c0000000-0000-0000-0000-000000000006','completed',9,true),
('a1000000-0000-0000-0000-000000000002','b0000000-0000-0000-0000-000000000001','completed',8,false),
('a1000000-0000-0000-0000-000000000002','b0000000-0000-0000-0000-000000000003','in_progress',NULL,false),
('a1000000-0000-0000-0000-000000000002','d0000000-0000-0000-0000-000000000007','planned',NULL,false),
('a1000000-0000-0000-0000-000000000002','d0000000-0000-0000-0000-000000000008','planned',NULL,false);

COMMIT;
