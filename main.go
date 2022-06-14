package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoInstance struct {
	Client *mongo.Client
	Db     *mongo.Database
}

var mg MongoInstance

const dbName = "fiber-hrms"
const mongoURI = "mongodb://user:pass@localhost:27027/" + dbName + "/?authSource=admin"

type Employee struct {
	ID     string  `json:"id,omitempty" bson:"_id,omitempty"`
	Name   string  `json:"name"`
	Salary float64 `json:"salary"`
	Age    float64 `json:"age"`
}

func Connect() error {
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		panic("Error conectando a db")
	}
	db := client.Database(dbName)
	mg = MongoInstance{
		Client: client,
		Db:     db,
	}

	return nil

}

func GetEmployee(c *fiber.Ctx) {
	query := bson.D{{}}
	cursor, err := mg.Db.Collection("employees").Find(c.Context(), query)
	if err != nil {
		c.Status(500).SendString(err.Error())
		return
	}

	var employees []Employee = make([]Employee, 0)

	if err = cursor.All(c.Context(), &employees); err != nil {
		c.Status(500).SendString(err.Error())
		return
	}
	c.JSON(employees)
}

func PostEmployee(c *fiber.Ctx) {
	collection := mg.Db.Collection("employees")
	employee := new(Employee)
	if err := c.BodyParser(employee); err != nil {
		c.Status(400).SendString(err.Error())
		return
	}
	employee.ID = ""
	insertionResult, err := collection.InsertOne(c.Context(), employee)
	if err != nil {
		c.Status(500).SendString(err.Error())
		return
	}
	filter := bson.D{{Key: "_id", Value: insertionResult.InsertedID}}
	createdRecord := collection.FindOne(c.Context(), filter)
	createdEmployee := &Employee{}
	createdRecord.Decode(createdEmployee)
	c.Status(200).JSON(createdEmployee)
}

func UpdateEmployee(c *fiber.Ctx) {
	idParam := c.Params("id")
	employeeID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.Status(400).SendString(err.Error())
		return
	}
	employee := new(Employee)
	if err := c.BodyParser(employee); err != nil {
		c.Status(400).SendString(err.Error())
		return
	}
	query := bson.D{{Key: "_id", Value: employeeID}}
	update := bson.D{
		{
			Key: "$set", Value: bson.D{
				{Key: "name", Value: employee.Name},
				{Key: "age", Value: employee.Age},
				{Key: "salary", Value: employee.Salary},
			},
		},
	}
	err = mg.Db.Collection("employees").FindOneAndUpdate(c.Context(), query, update).Err()
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.SendStatus(400)
			return
		}
		c.Status(500).SendString(err.Error())
		return
	}
	employee.ID = idParam
	c.Status(200).JSON(employee)

}

func DeleteEmployee(c *fiber.Ctx) {
	employeeID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		c.SendStatus(400)
		return
	}
	query := bson.D{{Key: "_id", Value: employeeID}}
	result, err := mg.Db.Collection("employees").DeleteOne(c.Context(), &query)
	if err != nil {
		c.SendStatus(500)
		return
	}
	if result.DeletedCount < 1 {
		c.SendStatus(404)
		return
	}
	c.Status(200).JSON("Record deleted")
}

func main() {

	if err := Connect(); err != nil {
		log.Fatal(err)
	}
	app := fiber.New()
	app.Get("/employee", GetEmployee)
	app.Post("/employee", PostEmployee)
	app.Put("/employee/:id", UpdateEmployee)
	app.Delete("/employee/:id", DeleteEmployee)
	log.Fatal(app.Listen(3456))
}
