-- -Clean up database droppings if the tests fail. This drops all of the tables


drop table apn            cascade;
drop table collection     cascade;
drop table device         cascade;
drop table device_lookup  cascade;
drop table downstream     cascade;
drop table firmware       cascade;
drop table firmware_image cascade;
drop table ghsession      cascade;
drop table ghstate        cascade;
drop table hordeuser      cascade;
drop table invite         cascade;
drop table lwm2mclient    cascade;
drop table magpie_data    cascade;
drop table member         cascade;
drop table nas            cascade;
drop table nasalloc       cascade;
drop table nonces         cascade;
drop table output         cascade;
drop table role           cascade;
drop table sequence       cascade;
drop table sessions       cascade;
drop table team           cascade;
drop table token          cascade;