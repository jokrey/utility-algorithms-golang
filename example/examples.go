package main

import (
	"utilitly_algorithms/network"
	"fmt"
)


func main() {
	do_mcnp_server_example()
}

func do_mcnp_server_example() {
	server := network.New_MCNP_Server(4567, func (conn network.MCNP_Connection) {
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
			fmt.Println("OK, received string")
			str, err := conn.Read_variable_chunk_utf8()
			fmt.Println("     here it is: ", str)
		}

		fmt.Println("====Succesfully handeled connection.")
	});

	server.RunListenerLoop()

	//time.Sleep(time.Second)
	//server.Close()
}