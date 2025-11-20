package service

import (
	"container/heap"
	"math"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"
	"github.com/s3nkyh/arcticeroute/models"
)

// ==============================
// ОСНОВНЫЕ СТРУКТУРЫ ДАННЫХ
// ==============================

// Route - маршрут с последовательностью точек
type Route struct {
	Points  []models.Point `json:"points"`  // Последовательность точек маршрута
	Length  float64        `json:"length"`  // Длина маршрута в метрах
	IsSafe  bool           `json:"is_safe"` // Безопасен ли маршрут
	Message string         `json:"message"` // Сообщение о маршруте
}

// NavNode - узел в навигационном графе
type NavNode struct {
	ID    string       `json:"id"`    // Уникальный идентификатор
	Point models.Point `json:"point"` // Координаты узла
	Type  string       `json:"type"`  // Тип: "port", "waypoint", "junction"
}

// NavEdge - ребро между узлами в графе
type NavEdge struct {
	From     string  `json:"from"`     // ID начального узла
	To       string  `json:"to"`       // ID конечного узла
	Distance float64 `json:"distance"` // Расстояние в метрах
	Cost     float64 `json:"cost"`     // Стоимость прохождения
}

// ==============================
// ГЕОГРАФИЧЕСКИЕ УТИЛИТЫ
// ==============================

// GeoUtils - утилиты для географических расчетов
type GeoUtils struct{}

// Distance вычисляет расстояние между двумя точками по формуле гаверсинусов
func (g *GeoUtils) Distance(p1, p2 models.Point) float64 {
	const R = 6371000 // Радиус Земли в метрах

	lat1 := p1.Lat * math.Pi / 180
	lon1 := p1.Lon * math.Pi / 180
	lat2 := p2.Lat * math.Pi / 180
	lon2 := p2.Lon * math.Pi / 180

	dLat := lat2 - lat1
	dLon := lon2 - lon1

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

// Bearing вычисляет начальный азимут между точками
func (g *GeoUtils) Bearing(p1, p2 models.Point) float64 {
	lat1 := p1.Lat * math.Pi / 180
	lon1 := p1.Lon * math.Pi / 180
	lat2 := p2.Lat * math.Pi / 180
	lon2 := p2.Lon * math.Pi / 180

	y := math.Sin(lon2-lon1) * math.Cos(lat2)
	x := math.Cos(lat1)*math.Sin(lat2) -
		math.Sin(lat1)*math.Cos(lat2)*math.Cos(lon2-lon1)

	bearing := math.Atan2(y, x) * 180 / math.Pi
	return math.Mod(bearing+360, 360)
}

// IntermediatePoint вычисляет промежуточную точку на дуге большого круга
func (g *GeoUtils) IntermediatePoint(p1, p2 models.Point, fraction float64) models.Point {
	lat1 := p1.Lat * math.Pi / 180
	lon1 := p1.Lon * math.Pi / 180
	lat2 := p2.Lat * math.Pi / 180
	lon2 := p2.Lon * math.Pi / 180

	δ := g.Distance(p1, p2) / 6371000 // Угловое расстояние в радианах

	a := math.Sin((1-fraction)*δ) / math.Sin(δ)
	b := math.Sin(fraction*δ) / math.Sin(δ)

	x := a*math.Cos(lat1)*math.Cos(lon1) + b*math.Cos(lat2)*math.Cos(lon2)
	y := a*math.Cos(lat1)*math.Sin(lon1) + b*math.Cos(lat2)*math.Sin(lon2)
	z := a*math.Sin(lat1) + b*math.Sin(lat2)

	lat := math.Atan2(z, math.Sqrt(x*x+y*y))
	lon := math.Atan2(y, x)

	return models.Point{
		Lat: lat * 180 / math.Pi,
		Lon: lon * 180 / math.Pi,
	}
}

// ==============================
// СИСТЕМА ОПРЕДЕЛЕНИЯ СУШИ
// ==============================

// LandDetector - определяет, находится ли точка на суше
type LandDetector struct {
	landPolygons []orb.Polygon // Полигоны суши
	region       orb.Bound     // Границы региона
}

// NewLandDetector создает детектор суши для региона
func NewLandDetector(minLat, maxLat, minLon, maxLon float64) *LandDetector {
	return &LandDetector{
		region: orb.Bound{
			Min: orb.Point{minLon, minLat},
			Max: orb.Point{maxLon, maxLat},
		},
		landPolygons: []orb.Polygon{},
	}
}

// AddLandPolygon добавляет полигон суши
func (ld *LandDetector) AddLandPolygon(points []models.Point) {
	ring := make(orb.Ring, len(points))
	for i, p := range points {
		ring[i] = orb.Point{p.Lon, p.Lat} // Note: orb uses [lon, lat]
	}
	// Замыкаем полигон
	ring = append(ring, ring[0])

	polygon := orb.Polygon{ring}
	ld.landPolygons = append(ld.landPolygons, polygon)
}

// IsLand определяет, находится ли точка на суше
func (ld *LandDetector) IsLand(point models.Point) bool {
	orbPoint := orb.Point{point.Lon, point.Lat}

	// Быстрая проверка границ
	if !ld.region.Contains(orbPoint) {
		return false
	}

	// Проверка всех полигонов суши
	for _, polygon := range ld.landPolygons {
		if planar.PolygonContains(polygon, orbPoint) {
			return true
		}
	}

	return false
}

// FindNearestWater находит ближайшую водную точку
func (ld *LandDetector) FindNearestWater(point models.Point, maxDistance float64) models.Point {
	if !ld.IsLand(point) {
		return point // Уже в воде
	}

	// Ищем в 8 направлениях
	directions := 8
	for distance := 0.1; distance <= maxDistance; distance += 0.1 {
		for i := 0; i < directions; i++ {
			angle := float64(i) * 360.0 / float64(directions)
			bearing := angle * math.Pi / 180.0

			// Вычисляем новую точку
			latRad := point.Lat * math.Pi / 180.0
			lonRad := point.Lon * math.Pi / 180.0

			newLat := math.Asin(math.Sin(latRad)*math.Cos(distance/6371.0) +
				math.Cos(latRad)*math.Sin(distance/6371.0)*math.Cos(bearing))

			newLon := lonRad + math.Atan2(
				math.Sin(bearing)*math.Sin(distance/6371.0)*math.Cos(latRad),
				math.Cos(distance/6371.0)-math.Sin(latRad)*math.Sin(newLat))

			testPoint := models.Point{
				Lat: newLat * 180.0 / math.Pi,
				Lon: newLon * 180.0 / math.Pi,
			}

			if !ld.IsLand(testPoint) {
				return testPoint
			}
		}
	}

	return point // Не нашли водную точку
}

// ==============================
// НАВИГАЦИОННЫЙ ГРАФ
// ==============================

// NavigationGraph - граф для поиска путей
type NavigationGraph struct {
	nodes map[string]*NavNode   // Узлы графа
	edges map[string][]*NavEdge // Исходящие ребра
	geo   *GeoUtils             // Географические утилиты
}

// NewNavigationGraph создает новый навигационный граф
func NewNavigationGraph() *NavigationGraph {
	return &NavigationGraph{
		nodes: make(map[string]*NavNode),
		edges: make(map[string][]*NavEdge),
		geo:   &GeoUtils{},
	}
}

// AddNode добавляет узел в граф
func (ng *NavigationGraph) AddNode(node *NavNode) {
	ng.nodes[node.ID] = node
	ng.edges[node.ID] = []*NavEdge{}
}

// AddEdge добавляет ребро между узлами
func (ng *NavigationGraph) AddEdge(fromID, toID string, costMultiplier float64) {
	from, fromExists := ng.nodes[fromID]
	to, toExists := ng.nodes[toID]

	if !fromExists || !toExists {
		return
	}

	distance := ng.geo.Distance(from.Point, to.Point)

	edge := &NavEdge{
		From:     fromID,
		To:       toID,
		Distance: distance,
		Cost:     distance * costMultiplier,
	}

	ng.edges[fromID] = append(ng.edges[fromID], edge)
}

// FindNearestNode находит ближайший узел к точке
func (ng *NavigationGraph) FindNearestNode(point models.Point, maxDistance float64) *NavNode {
	var nearest *NavNode
	minDist := math.MaxFloat64

	for _, node := range ng.nodes {
		dist := ng.geo.Distance(point, node.Point)
		if dist < minDist && dist <= maxDistance {
			minDist = dist
			nearest = node
		}
	}

	return nearest
}

// ==============================
// АЛГОРИТМ ПОИСКА ПУТИ (A*)
// ==============================

// pathNode - узел для алгоритма A*
type pathNode struct {
	nodeID    string
	cost      float64 // g(x) - стоимость от начала
	heuristic float64 // h(x) - эвристическая оценка
	total     float64 // f(x) = g(x) + h(x)
	parent    *pathNode
	index     int
}

// priorityQueue - приоритетная очередь для A*
type priorityQueue []*pathNode

func (pq priorityQueue) Len() int { return len(pq) }

func (pq priorityQueue) Less(i, j int) bool {
	return pq[i].total < pq[j].total
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *priorityQueue) Push(x interface{}) {
	n := len(*pq)
	node := x.(*pathNode)
	node.index = n
	*pq = append(*pq, node)
}

func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	node := old[n-1]
	node.index = -1
	*pq = old[0 : n-1]
	return node
}

// FindPath находит путь между узлами с помощью A*
func (ng *NavigationGraph) FindPath(startID, endID string) []*NavNode {
	if startID == endID {
		return []*NavNode{ng.nodes[startID]}
	}

	openSet := make(priorityQueue, 0)
	heap.Init(&openSet)

	startNode := &pathNode{
		nodeID:    startID,
		cost:      0,
		heuristic: ng.geo.Distance(ng.nodes[startID].Point, ng.nodes[endID].Point),
		total:     ng.geo.Distance(ng.nodes[startID].Point, ng.nodes[endID].Point),
	}
	heap.Push(&openSet, startNode)

	cameFrom := make(map[string]*pathNode)
	gScore := make(map[string]float64)
	gScore[startID] = 0

	for openSet.Len() > 0 {
		current := heap.Pop(&openSet).(*pathNode)

		if current.nodeID == endID {
			return ng.reconstructPath(cameFrom, current)
		}

		for _, edge := range ng.edges[current.nodeID] {
			tentativeG := gScore[current.nodeID] + edge.Cost

			if currentG, exists := gScore[edge.To]; !exists || tentativeG < currentG {
				cameFrom[edge.To] = current
				gScore[edge.To] = tentativeG

				heuristic := ng.geo.Distance(ng.nodes[edge.To].Point, ng.nodes[endID].Point)
				total := tentativeG + heuristic

				neighbor := &pathNode{
					nodeID:    edge.To,
					cost:      tentativeG,
					heuristic: heuristic,
					total:     total,
				}

				heap.Push(&openSet, neighbor)
			}
		}
	}

	return nil // Путь не найден
}

// reconstructPath восстанавливает путь из карты предков
func (ng *NavigationGraph) reconstructPath(cameFrom map[string]*pathNode, end *pathNode) []*NavNode {
	path := []*NavNode{ng.nodes[end.nodeID]}
	current := end

	for current.parent != nil {
		current = current.parent
		path = append([]*NavNode{ng.nodes[current.nodeID]}, path...)
	}

	return path
}

// ==============================
// ОСНОВНОЙ МАРШРУТИЗАТОР
// ==============================

// MarineRouter - основной класс маршрутизации
type MarineRouter struct {
	landDetector *LandDetector
	navGraph     *NavigationGraph
	geo          *GeoUtils
}

// NewMarineRouter создает новый маршрутизатор
func NewMarineRouter(minLat, maxLat, minLon, maxLon float64) *MarineRouter {
	return &MarineRouter{
		landDetector: NewLandDetector(minLat, maxLat, minLon, maxLon),
		navGraph:     NewNavigationGraph(),
		geo:          &GeoUtils{},
	}
}

// CalculateRoute вычисляет морской маршрут между точками
func (mr *MarineRouter) CalculateRoute(start, end models.Point) *Route {
	// 1. Проверяем и корректируем точки
	waterStart := mr.landDetector.FindNearestWater(start, 50.0) // Ищем в радиусе 50км
	waterEnd := mr.landDetector.FindNearestWater(end, 50.0)

	// 2. Находим ближайшие узлы графа
	startNode := mr.navGraph.FindNearestNode(waterStart, 50000) // В радиусе 50км
	endNode := mr.navGraph.FindNearestNode(waterEnd, 50000)

	if startNode == nil || endNode == nil {
		return &Route{
			Points:  []models.Point{start, end},
			IsSafe:  false,
			Message: "Не удалось найти подходящие навигационные точки",
		}
	}

	// 3. Ищем путь в графе
	pathNodes := mr.navGraph.FindPath(startNode.ID, endNode.ID)
	if pathNodes == nil {
		return &Route{
			Points:  []models.Point{start, end},
			IsSafe:  false,
			Message: "Маршрут не найден",
		}
	}

	// 4. Преобразуем в точки маршрута
	points := make([]models.Point, len(pathNodes))
	totalDistance := 0.0

	for i, node := range pathNodes {
		points[i] = node.Point
		if i > 0 {
			totalDistance += mr.geo.Distance(points[i-1], points[i])
		}
	}

	return &Route{
		Points:  points,
		Length:  totalDistance,
		IsSafe:  true,
		Message: "Маршрут успешно построен",
	}
}

// ==============================
// ПРИМЕР ИСПОЛЬЗОВАНИЯ
// ==============================

//func main() {
//	// Создаем маршрутизатор для региона Белого и Баренцева морей
//	router := NewMarineRouter(60.0, 70.0, 30.0, 50.0)
//
//	// Добавляем полигоны суши (упрощенно)
//	// Кольский полуостров
//	//kolaPeninsula := []models.Point{
//	//	{69.0, 32.0}, {68.5, 33.0}, {68.0, 34.0}, {67.5, 35.0},
//	//	{67.0, 36.0}, {66.5, 37.0}, {66.0, 38.0}, {66.5, 39.0},
//	//	{67.0, 40.0}, {68.0, 39.0}, {69.0, 38.0}, {69.5, 36.0},
//	//	{69.5, 34.0}, {69.0, 32.0},
//	//}
//	router.landDetector.AddLandPolygon(kolaPeninsula)
//
//	// Создаем навигационные узлы
//	nodes := []*NavNode{
//		{ID: "arh", Point: Point{64.54, 40.52}, Type: "port"},     // Архангельск
//		{ID: "mur", Point: Point{68.97, 33.07}, Type: "port"},     // Мурманск
//		{ID: "wp1", Point: Point{66.0, 38.0}, Type: "waypoint"},   // Промежуточная точка
//		{ID: "wp2", Point: Point{67.5, 36.5}, Type: "waypoint"},   // Промежуточная точка
//	}
//
//	for _, node := range nodes {
//		router.navGraph.AddNode(node)
//	}
//
//	// Создаем ребра (соединения между узлами)
//	router.navGraph.AddEdge("arh", "wp1", 1.0)
//	router.navGraph.AddEdge("wp1", "wp2", 1.0)
//	router.navGraph.AddEdge("wp2", "mur", 1.0)
//	router.navGraph.AddEdge("arh", "wp2", 1.2) // Альтернативный путь дороже
//
//	// Рассчитываем маршрут
//	start := Point{64.54, 40.52} // Архангельск
//	end := Point{68.97, 33.07}   // Мурманск
//
//	route := router.CalculateRoute(start, end)
//
//	// Выводим результат
//	fmt.Printf("Маршрут от Архангельска до Мурманска:\n")
//	fmt.Printf("Длина: %.2f км\n", route.Length/1000)
//	fmt.Printf("Безопасен: %t\n", route.IsSafe)
//	fmt.Printf("Сообщение: %s\n", route.Message)
//	fmt.Printf("Точек в маршруте: %d\n", len(route.Points))
//
//	for i, point := range route.Points {
//		fmt.Printf("%d: %.4f, %.4f\n", i, point.Lat, point.Lon)
//	}
//}
