package network

import (
	"net"
)

type MCNP_Connection struct {
	connection net.Conn
}

//Golang Constructor
func New_MCNP_Connection(connection net.Conn) MCNP_Connection {
	new := MCNP_Connection{connection}
	return new
}

func (c MCNP_Connection) Close() error {
	return c.connection.Close()
}


//SIMPLE API WRAPPER
func (c MCNP_Connection) Send_cause(cause int32) error {
	return Send_cause(c.connection, cause);
}
func (c MCNP_Connection) Read_cause() (int32, error) {
	return Read_cause(c.connection);
}
func (c MCNP_Connection) Send_fixed_chunk_byte(b byte) error {
	return Send_fixed_chunk_byte(c.connection, b);
}
func (c MCNP_Connection) Send_fixed_chunk_int16(i int16) error {
	return Send_fixed_chunk_int16(c.connection, i);
}
func (c MCNP_Connection) Send_fixed_chunk_int32(i int32) error {
	return Send_fixed_chunk_int32(c.connection, i);
}
func (c MCNP_Connection) Send_fixed_chunk_int64(i int64) error {
	return Send_fixed_chunk_int64(c.connection, i);
}
func (c MCNP_Connection) Send_fixed_chunk_float32(f float32) error {
	return Send_fixed_chunk_float32(c.connection, f);
}
func (c MCNP_Connection) Send_fixed_chunk_float64(d float64) error {
	return Send_fixed_chunk_float64(c.connection, d);
}
func (c MCNP_Connection) Send_fixed_chunk_bytes(bytes []byte) error {
	return Send_fixed_chunk_bytes(c.connection, bytes);
}
func (c MCNP_Connection) Read_fixed_chunk_byte() (byte, error) {
	return Read_fixed_chunk_byte(c.connection);
}
func (c MCNP_Connection) Read_fixed_chunk_int16() (int16, error) {
	return Read_fixed_chunk_int16(c.connection);
}
func (c MCNP_Connection) Read_fixed_chunk_int32() (int32, error) {
	return Read_fixed_chunk_int32(c.connection);
}
func (c MCNP_Connection) Read_fixed_chunk_int64() (int64, error) {
	return Read_fixed_chunk_int64(c.connection);
}
func (c MCNP_Connection) Read_fixed_chunk_float32() (float32, error) {
	return Read_fixed_chunk_float32(c.connection);
}
func (c MCNP_Connection) Read_fixed_chunk_float64() (float64, error) {
	return Read_fixed_chunk_float64(c.connection);
}
func (c MCNP_Connection) Read_fixed_chunk_bytes(bytesToRead int32) ([]byte, error) {
	return Read_fixed_chunk_bytes(c.connection, bytesToRead);
}
func (c MCNP_Connection) Start_chunk(chunk_size int64) error {
	return Start_chunk(c.connection, chunk_size);
}
func (c MCNP_Connection) Send_variable_chunk_utf8(s string) error {
	return Send_variable_chunk_utf8(c.connection, s);
}
func (c MCNP_Connection) Send_variable_chunk_bytearr(bytes []byte) error {
	return Send_variable_chunk_bytearr(c.connection, bytes);
}
func (c MCNP_Connection) Send_variable_chunk_from_file(filepath string) error {
	return Send_variable_chunk_from_file(c.connection, filepath)
}
func (c MCNP_Connection) Read_variable_chunk_utf8() (string, error) {
	return Read_variable_chunk_utf8(c.connection);
}
func (c MCNP_Connection) Read_variable_chunk_bytearr() ([]byte, error) {
	return Read_variable_chunk_bytearr(c.connection);
}
func (c MCNP_Connection) Read_variable_chunk_in_parts(received_part_callback func([]byte)) error {
	return Read_variable_chunk_in_parts(c.connection, received_part_callback);
}
func (c MCNP_Connection) Read_variable_chunk_into_file(filepath string) error {
	return Read_variable_chunk_into_file(c.connection, filepath)
}