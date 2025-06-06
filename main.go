package main

import (
  "context"
  "encoding/json"

  /*
    ALIASES AND GENERATED CODE
    The generated code is located within the /gen file. We're going to need some of the functions exported in there to implement our gRPC server. gRPC developers commonly alias these methods as 'pb' (Protocol Buffers) to indicate that this code is generated.
  */
  pb "go/tutorial/grpc/gen"
  "log"
  "net"
  "os"
  "time"

  "google.golang.org/grpc"
  "google.golang.org/grpc/codes"
  "google.golang.org/grpc/status"
)

/*
  TYPE EMBEDDING and UnimplementedBlogServer

  In go we can embed a type into another like this

  type server struct {
    pb.UnimplementedBlogServer
  }

  In this case, all the functions implemented by the UnimplementedBlogServer are automatically promoted to the server struct. If you inspect the gen/blog_grpc.pb.go file you'll find code that looks like this:

  <-- START CODE BLOCK -->
    type BlogServer interface {
      GetPosts(context.Context, *GetPostsRequest) (*Posts, error)
      CreatePost(context.Context, *CreatePostRequest) (*Post, error)
      mustEmbedUnimplementedBlogServer()
    }
    type UnimplementedBlogServer struct{}

    func (UnimplementedBlogServer) GetPosts(context.Context, *GetPostsRequest) (*Posts, error) {
      return nil, status.Errorf(codes.Unimplemented, "method GetPosts not implemented")
    }
    func (UnimplementedBlogServer) CreatePost(context.Context, *CreatePostRequest) (*Post, error) {
      return nil, status.Errorf(codes.Unimplemented, "method CreatePost not implemented")
    }
    func (UnimplementedBlogServer) mustEmbedUnimplementedBlogServer() {}
  <-- END CODE BLOCK -->

  As we can see from above the UnimplementedBlogServer has a default implementation for all the grpc calls defined in the protos, that is GetPosts and CreatePost, this is relevant because:
    - It makes the server forward compatible since we'll always have implementations for the methods defined in the proto file. If a client calls any of these methods, we'll raise a "not implemented" error.
    - By embedding the UnimplementedBlogServer, our server now complies with the BlogServer interface. As we'll see later in the main function, we need to register our server with pb.RegisterBlogServer(grpcServer, &server{}) and the second argument requires the server to implement the BlogServer interface.
*/

type server struct {
  pb.UnimplementedBlogServer
}

/*
  INTERFACES IN GO

  We briefly touched on interfaces above so it's good time to expand on the concept now. Interfaces in Go provide a way to specify the behavior of an object: if something can do this, then it can be used here. They define a contract of methods that a type must implement.

  A generic example:
  <-- START CODE BLOCK -->
    type Logger interface {
        Log(message string)
    }

    // FileLogger implements Logger
    type FileLogger struct {
        file string
    }

    func (f *FileLogger) Log(message string) {
        // Writes to file
    }

    // ConsoleLogger implements Logger
    type ConsoleLogger struct{}

    func (c *ConsoleLogger) Log(message string) {
        fmt.Println(message)
    }

    // Function that can work with any Logger
    func DoSomething(l Logger) {
        l.Log("Operation completed")
    }
  <-- END CODE BLOCK -->

  Both FileLogger and ConsoleLogger implement the Log method thus they both comply with the Logger interface. And because of that they both can be passed into the DoSomething method. Additionally, this allows for a form of duck-typing without the need of having inheritance.

  In our gRPC server, we use interfaces to define the contract that our server must implement, allowing different implementations while maintaining compatibility with the gRPC framework.
*/

var (
  filePath string = "posts.json"
)

/*
  As explained above now our server needs to override the GetPosts method to comply with the BlogServer interface. Notice how the function signature exactly matches the UnimplementedBlogServer including the arguments and return types.
*/
func (s *server) GetPosts(context.Context, *pb.GetPostsRequest) (*pb.Posts, error) {
  /*
    We need to leverage the types that protobuf generated for us. In this case we want to use Posts defined in the blog.pb.go

    type Posts struct {
      state         protoimpl.MessageState `protogen:"open.v1"`
      Posts         []*Post                `protobuf:"bytes,1,rep,name=posts,proto3" json:"posts,omitempty"`
      unknownFields protoimpl.UnknownFields
      sizeCache     protoimpl.SizeCache
    }

    We can see how the actual collection of post resize within the Posts property
  */

  posts := &pb.Posts{
    // We can use the make keyword to explicitly and easily generate an slice of Post references with an initial size 0.
    Posts: make([]*pb.Post, 0),
  }

  /*
   Notice that error handling is different than in the web version. Here we just return an error as opposed to having to write the error using the http writer.
  */
  if err := loadPost(posts); err != nil {
    return nil, err
  }

  for i := 0; i < len(posts.Posts); i++ {
    post := posts.Posts[i]
    post.ViewCount += 1
    post.LastViewed = time.Now().Format("2006-01-02")
  }

  if err := savePosts(posts); err != nil {
    return nil, status.Errorf(codes.Internal, "failed to save posts %v", err)
  }

  return posts, nil
}

// Similar to the above we need to comply exactly with the signature of the UnimplementedBlogServer
func (s *server) CreatePost(_ context.Context, req *pb.CreatePostRequest) (*pb.Post, error) {
  // We build a new post object by leveraging the stub definition.
  newPost := &pb.Post{
    Title:      req.GetTitle(),
    Content:    req.GetContent(),
    Author:     req.GetAuthor(),
    CreatedAt:  time.Now().Format("2006-01-02"),
    LastViewed: time.Now().Format("2006-01-02"),
    ViewCount:  0,
  }

  posts := &pb.Posts{
    Posts: make([]*pb.Post, 0),
  }

  if err := loadPost(posts); err != nil {
    return nil, status.Errorf(codes.Internal, "failed to load posts: %v\n", err)
  }

  posts.Posts = append(posts.Posts, newPost)

  if err := savePosts(posts); err != nil {
    return nil, status.Errorf(codes.Internal, "failed to save posts: %v", err)
  }

  return newPost, nil
}

func savePosts(posts *pb.Posts) error {
  data, err := json.MarshalIndent(posts.Posts, "", "  ")
  if err != nil {
    return err
  }

  return os.WriteFile(filePath, data, 0644)
}

func loadPost(posts *pb.Posts) error {
  data, err := os.ReadFile(filePath)

  if err != nil {
    return status.Errorf(codes.Internal, "failed to read posts file: %v\n", err)
  }

  var postsSlice []*pb.Post

  if err := json.Unmarshal(data, &postsSlice); err != nil {
    return status.Errorf(codes.Internal, "failed to parse post data %v\n", err)
  }

  posts.Posts = postsSlice

  return nil
}

func main() {
  // Contrary to the web example, in here we need to do a bit more setup
  // Set up TCP connection and start listening on port 3000
  lis, err := net.Listen("tcp", ":3000")

  if err != nil {
    log.Fatalf("failed to listen %s", err)
  }

  // Create the instance of the gRPC server
  grpcServer := grpc.NewServer()

  /*
    Register our server implementation with the gRPC server. As mentioned previously the RegisterBlogServer requires our server to implement the BlogServer interface

    type BlogServer interface {
      GetPosts(context.Context, *GetPostsRequest) (*Posts, error)
      CreatePost(context.Context, *CreatePostRequest) (*Post, error)
      mustEmbedUnimplementedBlogServer()
    }

    func RegisterBlogServer(s grpc.ServiceRegistrar, srv BlogServer) {
      if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
      t.testEmbeddedByValue()
      }
      s.RegisterService(&Blog_ServiceDesc, srv)
    }

    Showcasing why we embedded the pb.UnimplementedBlogServer into our server struct.
  */
  pb.RegisterBlogServer(grpcServer, &server{})

  // Finally we hook our server definitions to the tcp listener to start receiving requests.
  if err := grpcServer.Serve(lis); err != nil {
    log.Fatalf("Fail to server %s", err)
  }
}
