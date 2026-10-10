/*
 * Amaquet - A Newfoundcodes project.
 *
 * Copyright (C) 2026 Jonathan Eldy Baldivicio
 *
 * Author: Jonathan Eldy Baldivicio
 * Contact: jonathaneldy.baldivicio@newfoundcodes.com
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published
 * by the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

package types

import (
	"fmt"
	"math"
	"sort"
	"sync"
)

type GeoPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

func NewGeoPoint(lat, lon float64) (GeoPoint, error) {
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return GeoPoint{}, fmt.Errorf("invalid latitude/longitude")
	}
	return GeoPoint{lat, lon}, nil
}
func (p GeoPoint) DistanceMeters(q GeoPoint) float64 {
	const R = 6371008.8
	lat1, lat2 := p.Lat*math.Pi/180, q.Lat*math.Pi/180
	dlat := (q.Lat - p.Lat) * math.Pi / 180
	dlon := (q.Lon - p.Lon) * math.Pi / 180
	a := math.Sin(dlat/2)*math.Sin(dlat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dlon/2)*math.Sin(dlon/2)
	return 2 * R * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

type BoundingBox struct{ MinLat, MinLon, MaxLat, MaxLon float64 }

func NewBoundingBox(minLat, minLon, maxLat, maxLon float64) (BoundingBox, error) {
	if minLat > maxLat || minLon > maxLon {
		return BoundingBox{}, fmt.Errorf("invalid bounds")
	}
	if minLat < -90 || maxLat > 90 || minLon < -180 || maxLon > 180 {
		return BoundingBox{}, fmt.Errorf("bounds outside earth coordinates")
	}
	return BoundingBox{minLat, minLon, maxLat, maxLon}, nil
}
func (b BoundingBox) Contains(p GeoPoint) bool {
	return p.Lat >= b.MinLat && p.Lat <= b.MaxLat && p.Lon >= b.MinLon && p.Lon <= b.MaxLon
}

type Polygon struct {
	Points []GeoPoint `json:"points"`
}

func NewPolygon(points []GeoPoint) (Polygon, error) {
	if len(points) < 3 {
		return Polygon{}, fmt.Errorf("polygon needs at least 3 points")
	}
	return Polygon{Points: append([]GeoPoint(nil), points...)}, nil
}
func (p Polygon) Contains(pt GeoPoint) bool {
	inside := false
	j := len(p.Points) - 1
	for i := 0; i < len(p.Points); i++ {
		a, b := p.Points[i], p.Points[j]
		cross := (a.Lon > pt.Lon) != (b.Lon > pt.Lon)
		if cross {
			lat := (b.Lat-a.Lat)*(pt.Lon-a.Lon)/(b.Lon-a.Lon) + a.Lat
			if pt.Lat < lat {
				inside = !inside
			}
		}
		j = i
	}
	return inside
}
func (p Polygon) Bounds() BoundingBox {
	b := BoundingBox{MinLat: p.Points[0].Lat, MaxLat: p.Points[0].Lat, MinLon: p.Points[0].Lon, MaxLon: p.Points[0].Lon}
	for _, x := range p.Points[1:] {
		if x.Lat < b.MinLat {
			b.MinLat = x.Lat
		}
		if x.Lat > b.MaxLat {
			b.MaxLat = x.Lat
		}
		if x.Lon < b.MinLon {
			b.MinLon = x.Lon
		}
		if x.Lon > b.MaxLon {
			b.MaxLon = x.Lon
		}
	}
	return b
}

type GeoItem struct {
	ID       string            `json:"id"`
	Point    GeoPoint          `json:"point"`
	Metadata map[string]string `json:"metadata,omitempty"`
}
type GeoResult struct {
	GeoItem
	DistanceMeters float64 `json:"distance_meters"`
}
type geoCell struct{ lat, lon int }

const geoCellDegrees = 0.25

func geoCellFor(p GeoPoint) geoCell {
	return geoCell{int(math.Floor((p.Lat + 90) / geoCellDegrees)), int(math.Floor((p.Lon + 180) / geoCellDegrees))}
}

type GeoSpatialIndex struct {
	mu       sync.RWMutex
	items    map[string]GeoItem
	cells    map[geoCell]map[string]struct{}
	itemCell map[string]geoCell
}

func NewGeoSpatialIndex() *GeoSpatialIndex {
	return &GeoSpatialIndex{items: map[string]GeoItem{}, cells: map[geoCell]map[string]struct{}{}, itemCell: map[string]geoCell{}}
}
func (g *GeoSpatialIndex) Add(id string, p GeoPoint, metadata map[string]string) {
	cp := map[string]string{}
	for k, v := range metadata {
		cp[k] = v
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if old, ok := g.itemCell[id]; ok {
		delete(g.cells[old], id)
		if len(g.cells[old]) == 0 {
			delete(g.cells, old)
		}
	}
	c := geoCellFor(p)
	if g.cells[c] == nil {
		g.cells[c] = map[string]struct{}{}
	}
	g.cells[c][id] = struct{}{}
	g.itemCell[id] = c
	g.items[id] = GeoItem{id, p, cp}
}
func (g *GeoSpatialIndex) Remove(id string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	_, ok := g.items[id]
	if !ok {
		return false
	}
	if c, yes := g.itemCell[id]; yes {
		delete(g.cells[c], id)
		if len(g.cells[c]) == 0 {
			delete(g.cells, c)
		}
		delete(g.itemCell, id)
	}
	delete(g.items, id)
	return true
}
func cellRange(box BoundingBox) (int, int, int, int) {
	a := geoCellFor(GeoPoint{box.MinLat, box.MinLon})
	b := geoCellFor(GeoPoint{box.MaxLat, box.MaxLon})
	return a.lat, b.lat, a.lon, b.lon
}
func (g *GeoSpatialIndex) candidateIDsLocked(box BoundingBox) []string {
	minLat, maxLat, minLon, maxLon := cellRange(box)
	cells := (maxLat - minLat + 1) * (maxLon - minLon + 1)
	if cells > 100000 {
		out := make([]string, 0, len(g.items))
		for id := range g.items {
			out = append(out, id)
		}
		return out
	}
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for la := minLat; la <= maxLat; la++ {
		for lo := minLon; lo <= maxLon; lo++ {
			for id := range g.cells[geoCell{la, lo}] {
				if _, ok := seen[id]; !ok {
					seen[id] = struct{}{}
					out = append(out, id)
				}
			}
		}
	}
	return out
}
func (g *GeoSpatialIndex) WithinRadius(center GeoPoint, radiusMeters float64, limit int) []GeoResult {
	if radiusMeters < 0 {
		return nil
	}
	latDelta := radiusMeters / 111320.0
	cos := math.Cos(center.Lat * math.Pi / 180)
	if math.Abs(cos) < 0.01 {
		cos = 0.01
	}
	lonDelta := radiusMeters / (111320.0 * math.Abs(cos))
	minLat := math.Max(-90, center.Lat-latDelta)
	maxLat := math.Min(90, center.Lat+latDelta)
	minLon := math.Max(-180, center.Lon-lonDelta)
	maxLon := math.Min(180, center.Lon+lonDelta)
	g.mu.RLock()
	defer g.mu.RUnlock()
	ids := g.candidateIDsLocked(BoundingBox{minLat, minLon, maxLat, maxLon})
	out := make([]GeoResult, 0)
	for _, id := range ids {
		it := g.items[id]
		d := center.DistanceMeters(it.Point)
		if d <= radiusMeters {
			out = append(out, GeoResult{it, d})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DistanceMeters < out[j].DistanceMeters })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}
func (g *GeoSpatialIndex) WithinBox(box BoundingBox) []GeoItem {
	g.mu.RLock()
	defer g.mu.RUnlock()
	ids := g.candidateIDsLocked(box)
	out := make([]GeoItem, 0)
	for _, id := range ids {
		it := g.items[id]
		if box.Contains(it.Point) {
			out = append(out, it)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
func (g *GeoSpatialIndex) WithinPolygon(poly Polygon) []GeoItem {
	box := poly.Bounds()
	g.mu.RLock()
	defer g.mu.RUnlock()
	ids := g.candidateIDsLocked(box)
	out := make([]GeoItem, 0)
	for _, id := range ids {
		it := g.items[id]
		if poly.Contains(it.Point) {
			out = append(out, it)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
