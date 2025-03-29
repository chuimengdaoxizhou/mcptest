package allgrpc

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "mcpclient/allgrpc/allproto"
)

func Getdata(prompt string) string {
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return "连接服务器失败"
	}
	defer conn.Close()

	client := pb.NewDataManagementClient(conn)

	var req pb.Request
	req.Prompt = prompt
	resp, err := client.GetDatabyPrompt(context.Background(), &req)
	if err != nil {
		return "调用服务端失败"
	}
	return resp.Answer
}

func Updata(filepath string) string {
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err.Error()
	}
	defer conn.Close()

	client := pb.NewDataManagementClient(conn)

	var req pb.Request
	req.Prompt = filepath
	resp, err := client.Updatabypath(context.Background(), &req)
	if err != nil {
		return err.Error()
	}
	return resp.Answer
}
