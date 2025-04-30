package main

import (
  "context"
  "fmt"
  pb "go/tutorial/grpc/gen"
  "log"
  "time"

  "google.golang.org/grpc"
  "google.golang.org/grpc/credentials/insecure"
)

/*
  protoc also generates client code that we can use to test our grpc server. We'll be using this example to test both the CreatePost and GetPosts methods.
*/

func main() {
  /*
   We create a new connection and bind it to localhost:3000 (the same port used on the server side).

   WithTransportCredentials configures the security settings for the gRPC connection:
    - insecure.NewCredentials() creates an insecure channel (no SSL/TLS)
    - Only suitable for development/testing
    - For production, use proper TLS credentials:
    creds := credentials.NewTLS(&tls.Config{...})
    conn := grpc.Dial(address, grpc.WithTransportCredentials(creds))
  */
  conn, err := grpc.NewClient("localhost:3000", grpc.WithTransportCredentials(insecure.NewCredentials()))

  if err != nil {
    log.Fatalf("failed to connect to grpc server")
  }

  // Make sure that we close the connection at the end of execution.
  defer conn.Close()

  // Create a new instance of the client using the previously created connection.
  c := pb.NewBlogClient(conn)

  /*
   CONTEXT IN GO

   Context provides a way to carry deadlines, cancellation signals, and request-scoped values across API boundaries. In this client:

   1. Creation and Timeout:
     This creates a context with 1 second timeout. The context countdown starts as soon as the context gets created.
     ctx, cancel := context.WithTimeout(context.Background(), time.Second)
     defer cancel()  // Always call cancel to prevent resource leaks

   2. Usage:
     Functions thats use this context will only run for the time remaining in the context.
     - For example, if CreatePost takes 0.7s, GetPosts only has 0.3s remaining (assuming no meaningful amount of time has been spent on the main function)
     - When context expires, the operation that is using the context is cancelled and an error is returned

   3. Best Practices:
     - Create separate contexts for independent operations
     - Pass context as first parameter to functions
     - Don't store context in structs
     - Use for request-scoped data only

   Context is not specific to gRPC - it's a standard Go feature used across the ecosystem for managing timeouts, cancellation, and request-scoped values in APIs, database calls, HTTP requests, and more.
  */
  ctx, cancel := context.WithTimeout(context.Background(), time.Second)
  defer cancel()

  // Create a new post first using the gRPC method
  newPost := &pb.CreatePostRequest{
    Title:   "My very first gRPC Post",
    Content: "This is  test post with gRPC",
    Author:  "gRPC client",
  }

  // We call the client CreatePost function passing context and the CreatePostRequest
  post, err := c.CreatePost(ctx, newPost)

  if err != nil {
    /*
     log.Fatalf:
     - The message is logged
     - Deferred functions are executed
     - Then os.Exit(1) is called
    */
    log.Fatalf("could not create post: %v", err)
  }

  fmt.Printf("Created Post: %v", post)

  // We call the client GetPosts function passing context and the GetPostsRequest
  posts, err := c.GetPosts(ctx, &pb.GetPostsRequest{})

  if err != nil {
    log.Fatalf("could not get posts: %v", err)
  }

  fmt.Println("\n All Posts:")

  // We printout the posts to std out for confirmation.
  for _, p := range posts.Posts {
    fmt.Printf("Title: %s\nAuthor: %s\nContent: %s\nView Count: %d\n\n",
      p.GetTitle(),
      p.GetAuthor(),
      p.GetContent(),
      p.GetViewCount(),
    )
  }
}
