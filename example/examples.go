package main

import (
	"fmt"
	"github.com/jokrey/utility-algorithms-golang/network/mcnp"
	"github.com/jokrey/utility-algorithms-golang/stringencoder"
	"time"
)

func main() {
	//do_tag_system_test()

	//do_mcnp_server_example()
	//do_mcnp_client_example()

	//self contained example run.
	go do_mcnp_server_example()
	time.Sleep(time.Second*10)
	do_mcnp_client_example()
	for true {}// Without goroutine synchronization for simplicity. Has to be manually shut down.
}

//Error handling should obviously be done differently in reality, but for this example its sufficient
func do_mcnp_client_example() {
	client, err := mcnp.New_MCNP_Client_Connection(
		"localhost",4567, time.Second)
	if err != nil {panic(err)}

	client.Send_cause(1)

	var b byte = 12
	var i16 int16 = 1231
	var i32 int32 = 123
	var i64 int64 = 234234
	var f32 float32 = 123.123
	var f64 float64 = 1234.1234
	err = client.Send_fixed_chunk_byte(b)
	if err != nil {panic(err)}
	err = client.Send_fixed_chunk_int16(i16)
	if err != nil {panic(err)}
	err = client.Send_fixed_chunk_int32(i32)
	if err != nil {panic(err)}
	err = client.Send_fixed_chunk_int64(i64)
	if err != nil {panic(err)}
	err = client.Send_fixed_chunk_float32(f32)
	if err != nil {panic(err)}
	err = client.Send_fixed_chunk_float64(f64)
	if err != nil {panic(err)}

	i32_back, err := client.Read_fixed_chunk_int32()
	if err != nil {panic(err)}
	fmt.Println("Received i32 back",i32_back)

	bytes := []byte{12,123,4,23,54,233,34,3,2,1}//Add or remove any number of bytes here
	err = client.Send_variable_chunk_bytearr(bytes)
	if err != nil {panic(err)}

	bytes1 := []byte{12,123,4,23,54,233,34}//Add or remove any number of bytes here
	bytes2 := []byte{12,68,4,23,3,233,34, 1,2,3}//Add or remove any number of bytes here
	err = client.Start_chunk(int64(len(bytes1) + len(bytes2)))
	if err != nil {panic(err)}
	err = client.Send_chunk_part(bytes1)
	if err != nil {panic(err)}
	err = client.Send_chunk_part(bytes2)
	if err != nil {panic(err)}

	utf8_exmpl := "This is a what, what. This is a test."
	err = client.Send_variable_chunk_utf8(utf8_exmpl)
	if err != nil {panic(err)}

	//Commented out because it requires a file
	//user, err := user.Current()
	//if err != nil { panic(err) }
	//output_path := user.HomeDir + "/Desktop/example_to_send.txt"
	//err = client.Send_variable_chunk_from_file(output_path)
	//if err != nil {panic(err)}
}

//Error handling should obviously be done differently in reality, but for this example its sufficient
func do_mcnp_server_example() {
	server := mcnp.New_MCNP_Server(4567, func (conn mcnp.MCNP_Connection) {
		//defer conn.Close()// NOT NECESSARY. THE SERVER DOES THIS FOR US
		fmt.Println("====Handeling new connection.")
		cause, err := conn.Read_cause()
		fmt.Println("Received cause:",cause)
		if err != nil { panic(err) }
		switch cause {
		case -11:
			fmt.Println("Well. Apparently the client wants to disconnect already.")
		case 1:
			fmt.Println("OK, the client got something for us (all the fixed chunk types, in order):")
			b,err := conn.Read_fixed_chunk_byte()
			if err != nil { panic(err) }
			i16,err := conn.Read_fixed_chunk_int16()
			if err != nil { panic(err) }
			i32,err := conn.Read_fixed_chunk_int32()
			if err != nil { panic(err) }
			i64,err := conn.Read_fixed_chunk_int64()
			if err != nil { panic(err) }
			f32,err := conn.Read_fixed_chunk_float32()
			if err != nil { panic(err) }
			f64,err := conn.Read_fixed_chunk_float64()
			if err != nil { panic(err) }
			fmt.Println("OK received all expected fixed chunk types:")
			fmt.Println("    byte:(",b,"), int16:(",i16,"), int32:(",i32,"), int64:(",i64,"), float32:(",f32,"), float64:(",f64,")")

			fmt.Println("OK, now the client actually excepts something from us for some reason. An int32. We will just return what it send us.")
			err = conn.Send_fixed_chunk_int32(i32)
			if err != nil { panic(err) }
			fmt.Println("Send ",i32)

			fmt.Println("OK, the client got something else for us (a small byte array, but without fixed length):")
			bytes, err := conn.Read_variable_chunk_bytearr()
			if err != nil { panic(err) }
			fmt.Println("OK, received an array of length: ", len(bytes))
			fmt.Println("     here it is: ", bytes)

			fmt.Println("OK, the client got something else for us (a small byte array, but without fixed length):")
			bytes, err = conn.Read_variable_chunk_bytearr()
			if err != nil { panic(err) }
			fmt.Println("OK, received an array of length: ", len(bytes))
			fmt.Println("     here it is: ", bytes)

			fmt.Println("OK, now the client got a string for us")
			str, err := conn.Read_variable_chunk_utf8()
			if err != nil { panic(err) }
			fmt.Println("OK, received string")
			fmt.Println("     here it is: ", str)

			//Commented out because it requires a file
			//fmt.Println("OK, now the client got a file for us.")
			//user, err := user.Current()
			//if err != nil { panic(err) }
			//output_path := user.HomeDir + "/Desktop/mcnp_received.txt"
			//err = conn.Read_variable_chunk_into_file(output_path)
			//if err != nil { panic(err) }
			//fmt.Println("OK, wrote file to: ", output_path)
		default:
			fmt.Println("Cause not recognised. Please check the clients configuration, it doesn't seem to know what we want.")
		}

		fmt.Println("====Succesfully handeled connection.")
	})

	err := server.RunListenerLoop()
	if err != nil { panic(err) }

	//Example of doing it in a new goroutine:
	//go server.RunListenerLoop()
	//time.Sleep(time.Second)
	//server.Close()
}

func do_tag_system_test() {
	//order, values and  don't matter
	//just keep in mind that tagged values have to be written into the working string, before being read out.
	var encoder stringencoder.AdvancedStringEncoder = stringencoder.New_AdvancedStringEncoder("")
	encoder.AddEntry("tag1", "Whatup")
	encoder.AddEntry_bool("tag2", true)
	encoder.AddEntry_i64("tag3", 1234123)
	encodable := TestEncodable{123,true}
	encoder.AddEntry_encodable("tag4", &encodable)
	encodedString := encoder.GetEncodedString()
	fmt.Println("encodedString: "+encodedString)

	decoder := stringencoder.New_AdvancedStringEncoder(encodedString)
	entry_tag1, err := decoder.GetEntry("tag1") //error handeling available
	if err!=nil {panic(err)}
	fmt.Println("at tag1: "+entry_tag1)
	fmt.Println("at tag2: ",decoder.DeleteEntry_bool("tag2"))
	fmt.Println("at tag3: ",decoder.GetEntry_i64("tag3")) //deleting can increase performance over time, because less items will have to iteratet
	newTestEncodable := TestEncodable{}
	decoder.NewEncodableFromEntry("tag4", &newTestEncodable)
	fmt.Println("at tag4: ",newTestEncodable)
	fmt.Println("encodedString: "+decoder.GetEncodedString())
}

//implements EncodableAsString interface
type TestEncodable struct {
	test1 int32
	test2 bool
}
func (te TestEncodable) GetEncodedString() string {
	encoder := stringencoder.New_AdvancedStringEncoder("")
	encoder.AddEntry_i64("1", int64(te.test1))
	encoder.AddEntry_bool("2", te.test2)
	return encoder.GetEncodedString()
}
func (dummy *TestEncodable) NewFromEncodedString(encoded string) {
	decoder := stringencoder.New_AdvancedStringEncoder(encoded)
	dummy.test1 = int32(decoder.GetEntry_i64("1"))
	dummy.test2 = decoder.GetEntry_bool("2")
}
