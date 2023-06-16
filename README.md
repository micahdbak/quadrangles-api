# API for quadrangles

This API is dependant on the existence of a PostgreSQL server accessible on the `localhost`.
A `quadrangles` database must be present, and the following tables must be created:

```sql
CREATE TABLE files (
    fid   serial primary key,
    ctype varchar(127),
    name  varchar(127),
    time  bigint
);

CREATE TABLE posts (
    pid   serial primary key,
    fid   serial references files(fid),
    topic varchar(4),
    text  varchar(2000),
    time  bigint
);

CREATE TABLE comments (
    cid  serial primary key,
    pid  serial references posts(pid),
    text varchar(2000),
    time bigint
);
```
