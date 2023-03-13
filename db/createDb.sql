create table zfc_user (
    id                         serial PRIMARY KEY, 
    nftkeyme_id                varchar(64) not null,
    nftkeyme_access_token      varchar(128),
    nftkeyme_refresh_token     varchar(128),
    last_asset_check_time      timestamptz,
    UNIQUE(nftkeyme_id)
);

-- import nfts after creating, then set types
create table nft (
    id              serial PRIMARY KEY, 
    name            varchar(32) not null,
    nft_type        varchar(32) not null,
    wins            integer DEFAULT 0,
    loses           integer DEFAULT 0,
    UNIQUE(name)
);

create table zfc_user_nft (
    zfc_user_id             int,
    nft_id                  int,
    amount_ada              integer,
    listed_date             timestamp, 
    UNIQUE(zfc_user_id,nft_id),
    CONSTRAINT FK_zfc_user_id FOREIGN KEY(zfc_user_id) REFERENCES zfc_user(id),
    CONSTRAINT FK_nft_id FOREIGN KEY(nft_id) REFERENCES nft(id)
);

create table fight (
    id                         serial PRIMARY KEY,
    hunter_user_id             integer not null,
    hunter_nft_id              integer not null,
    hunter_amount_ada          integer not null,
    hunter_send_address        varchar(128),
    zombie_user_id             integer not null,
    zombie_nft_id              integer not null,
    zombie_amount_ada          integer not null,
    zombie_send_address        varchar(128),
    payment_amount_lovelace    bigint not null,
    payment_address            varchar(128) not null,     
    status                     varchar(64) not null,
    incoming_utxo              varchar(128),
    incoming_utxo_index        integer,
    minting_user_id            integer not null,
    ipfs_fight                 varchar(128),
    created_date               timestamptz DEFAULT NOW(),
    minted_date                timestamptz,
    tx_id                      varchar(128),  
    background                 varchar(64),
    zhLifeBar                  integer,
    zcLifeBar                  integer,
    hunter_record              varchar(64),
    zombie_record              varchar(64),
    hunter_ko                  boolean,
    zombie_ko                  boolean,
    hunter_beatup              boolean,
    zombie_beatup              boolean,
    tweet_id                   varchar(128),
    collection                 varchar(64) DEFAULT 'Zombie Fight Club',
    site                       varchar(64) DEFAULT 'https://zombiechains.io/',
    twitter                    varchar(64) DEFAULT 'https://twitter.com/ZombieChains',
    copyright                  varchar(64) DEFAULT '2022 Zombie Chains',
    UNIQUE(incoming_utxo, incoming_utxo_index),
    CONSTRAINT FK_hunter_user_id FOREIGN KEY(hunter_user_id) REFERENCES zfc_user(id),
    CONSTRAINT FK_zombie_user_id FOREIGN KEY(zombie_user_id) REFERENCES zfc_user(id),
    CONSTRAINT FK_minting_user_id FOREIGN KEY(minting_user_id) REFERENCES zfc_user(id),
    CONSTRAINT FK_hunter_nft_id FOREIGN KEY(hunter_nft_id) REFERENCES nft(id),
    CONSTRAINT FK_zombie_nft_id FOREIGN KEY(zombie_nft_id) REFERENCES nft(id)
);

create table zfc_alien (
    id                         serial PRIMARY KEY,
    fight_id                   integer,
    ipfs_hash                  varchar(64),
    name                       varchar(32) not null,
    readable_name              varchar(64) not null,
    background                 varchar(64) not null,
    skin                       varchar(64) not null,
    clothes                    varchar(64) not null,
    eyes                       varchar(64) not null,
    mouth                      varchar(64) not null,
    hand                       varchar(64) not null,
    hat                        varchar(64) not null,
    collection                 varchar(64) DEFAULT 'Zombie Fight Club Aliens',
    site                       varchar(64) DEFAULT 'https://zombiechains.io/',
    twitter                    varchar(64) DEFAULT 'https://twitter.com/ZombieChains',
    copyright                  varchar(64) DEFAULT '2022 Zombie Chains',
    CONSTRAINT FK_fight_id FOREIGN KEY(fight_id) REFERENCES fight(id),
    UNIQUE(background,skin,clothes,eyes,mouth,hand,hat)
);