package main

import "math/rand"

// Predefined routes for drivers around Hyderabad (Madhapur / Hitech City / Jubilee Hills / Banjara Hills).
var PredefinedRoutes = [][][]float64{
	{
		{17.4504, 78.3808},
		{17.4486, 78.3908},
		{17.4435, 78.3972},
		{17.4401, 78.3915},
	},
	{
		{17.4482, 78.3915},
		{17.4445, 78.3850},
		{17.4390, 78.3810},
		{17.4355, 78.3865},
		{17.4320, 78.3920},
		{17.4295, 78.3980},
		{17.4270, 78.4040},
	},
	{
		{17.4326, 78.4071},
		{17.4270, 78.4120},
		{17.4210, 78.4180},
		{17.4165, 78.4240},
		{17.4120, 78.4300},
		{17.4080, 78.4360},
	},
	{
		{17.4156, 78.4482},
		{17.4120, 78.4550},
		{17.4080, 78.4620},
		{17.4035, 78.4680},
		{17.3990, 78.4740},
		{17.3950, 78.4800},
	},
}

func GenerateRandomPlate() string {
	letters := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	plate := "TS09"
	for i := 0; i < 2; i++ {
		plate += string(letters[rand.Intn(len(letters))])
	}
	plate += string(rune('0' + rand.Intn(10)))
	plate += string(rune('0' + rand.Intn(10)))
	plate += string(rune('0' + rand.Intn(10)))
	plate += string(rune('0' + rand.Intn(10)))
	return plate
}
