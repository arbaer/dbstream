create table mawi_wide (serial_time int4,
	server_port int4,
	user_port int4,
	server_ip inet,
	user_ip inet,
	protocol int2,
	ipv4_protocol int2,
	flags int4,
	up_phy_pkts int8,
	up_ipv4_pkts int8,
	up_phy_bytes int8,
	up_ipv4_pdu_bytes int8,
	up_ipv4_sdu_bytes int8
);
