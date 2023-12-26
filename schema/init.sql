
create table if not exists users(
    id int auto_increment primary key,
    user_name varchar(255) not null,
    user_surname varchar(255) not null,
    user_login varchar(10) not null,
    user_password varchar(10) not null
);

create table if not exists conferences(
    conference_id int auto_increment primary key,
    conference_name varchar(255) not null
);

create table if not exists reports(
    report_id int auto_increment primary key,
    report_name varchar(255) not null,
    id int,
    conference_id int,
    constraint fk_report_author foreign key (id) references users(id),
    constraint fk_conference foreign key (conference_id) references conferences(conference_id)
);

insert into users (id, user_name, user_surname, user_login, user_password) VALUES (10000, 'admin','admin','log1','pass1');
insert into reports (report_name, id) VALUES ("arch_lab_kursach", 10000);
insert into conferences (conference_name) VALUES ("arch_lab_conference");