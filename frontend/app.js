const map = L.map('map').setView([75.0, 40.0], 4);

L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
    attribution: '© OpenStreetMap contributors'
}).addTo(map);

const shipIcon = L.divIcon({
    className: 'ship-marker',
    html: '🚢',
    iconSize: [30, 30],
    iconAnchor: [15, 15]
});

const glacierIcon = L.divIcon({
    className: 'glacier-marker',
    html: '🧊',
    iconSize: [25, 25],
    iconAnchor: [12, 12]
});

let shipMarkers = [];
let glacierMarkers = [];

const API_BASE = 'http://localhost:8080';

async function loadData() {
    const button = document.getElementById('loadData');
    button.disabled = true;
    button.textContent = 'Loading...';

    try {
        clearMarkers();

        console.log('🚢 Fetching ships...');
        const shipsResponse = await fetch(`${API_BASE}/api/ships`);

        if (!shipsResponse.ok) {
            throw new Error(`Ships HTTP error! status: ${shipsResponse.status}`);
        }

        const ships = await shipsResponse.json();
        console.log('✅ Raw ships data:', ships);

        console.log('🧊 Fetching glaciers...');
        const glaciersResponse = await fetch(`${API_BASE}/api/glaciers?bbox=65,30,90,180`);

        if (!glaciersResponse.ok) {
            throw new Error(`Glaciers HTTP error! status: ${glaciersResponse.status}`);
        }

        const glaciers = await glaciersResponse.json();
        console.log('✅ Raw glaciers data:', glaciers);

        addShipsToMap(ships);
        addGlaciersToMap(glaciers);

        document.getElementById('shipCount').textContent = `Ships: ${shipMarkers.length}`;
        document.getElementById('glacierCount').textContent = `Glaciers: ${glacierMarkers.length}`;

        const allMarkers = [...shipMarkers, ...glacierMarkers];
        if (allMarkers.length > 0) {
            const group = new L.featureGroup(allMarkers);
            map.fitBounds(group.getBounds().pad(0.1));
            console.log(`🗺️ Map fitted to ${allMarkers.length} markers`);
        } else {
            console.warn('❌ No valid markers to display');
        }

    } catch (error) {
        console.error('❌ Error loading data:', error);
        alert('Error loading data: ' + error.message);
    } finally {
        button.disabled = false;
        button.textContent = 'Load Ships & Glaciers';
    }
}

function addShipsToMap(ships) {
    if (!ships || !Array.isArray(ships)) {
        console.error('❌ Invalid ships data:', ships);
        return;
    }

    let validShips = 0;
    let invalidShips = 0;

    ships.forEach((ship, index) => {
        console.log(`🔍 Ship ${index}:`, ship);

        const lat = ship.latitude || ship.Latitude || ship.lat;
        const lon = ship.longitude || ship.Longitude || ship.lon;
        const name = ship.name || ship.Name || `Ship ${ship.mmsi || ship.MMSI}`;
        const mmsi = ship.mmsi || ship.MMSI;

        console.log(`📡 Ship ${index} coordinates:`, { lat, lon, name, mmsi });

        if (!isValidCoordinate(lat, lon)) {
            console.warn(`❌ Invalid ship coordinates: ${name} (${lat}, ${lon})`);
            invalidShips++;
            return;
        }

        try {
            const marker = L.marker([lat, lon], { icon: shipIcon })
                .addTo(map)
                .bindPopup(`
                    <div class="ship-popup">
                        <h3>${name}</h3>
                        <div class="popup-info">
                            <strong>MMSI:</strong> ${mmsi || 'N/A'}<br>
                            <strong>Position:</strong> ${lat.toFixed(4)}, ${lon.toFixed(4)}
                        </div>
                    </div>
                `);

            marker.on('click', () => {
                document.getElementById('selectedInfo').innerHTML = `
                    <h4>🚢 ${name}</h4>
                    <p><strong>MMSI:</strong> ${mmsi || 'N/A'}</p>
                    <p><strong>Latitude:</strong> ${lat.toFixed(6)}</p>
                    <p><strong>Longitude:</strong> ${lon.toFixed(6)}</p>
                `;
            });

            shipMarkers.push(marker);
            validShips++;
            console.log(`✅ Added ship: ${name}`);

        } catch (error) {
            console.error(`❌ Error adding ship ${name}:`, error);
            invalidShips++;
        }
    });

    console.log(`📊 Ships summary: ${validShips} valid, ${invalidShips} invalid`);
}

function addGlaciersToMap(glaciers) {
    if (!glaciers || !Array.isArray(glaciers)) {
        console.error('❌ Invalid glaciers data:', glaciers);
        return;
    }

    let validGlaciers = 0;
    let invalidGlaciers = 0;

    glaciers.forEach((glacier, index) => {
        if (index < 5) {
            console.log(`🔍 Glacier ${index}:`, glacier);
        }

        const lat = glacier.latitude || glacier.Latitude || glacier.lat;
        const lon = glacier.longitude || glacier.Longitude || glacier.lon;
        const name = glacier.name || glacier.Name || `Glacier ${glacier.id || glacier.ID}`;
        const type = glacier.type || glacier.Type || 'Unknown';
        const id = glacier.id || glacier.ID;

        if (!isValidCoordinate(lat, lon)) {
            invalidGlaciers++;
            if (invalidGlaciers <= 5) {
                console.warn(`❌ Invalid glacier coordinates: ${name} (${lat}, ${lon})`);
            }
            return;
        }

        try {
            const marker = L.marker([lat, lon], { icon: glacierIcon })
                .addTo(map)
                .bindPopup(`
                    <div class="glacier-popup">
                        <h3>${name}</h3>
                        <div class="popup-info">
                            <strong>Type:</strong> ${type}<br>
                            <strong>Position:</strong> ${lat.toFixed(4)}, ${lon.toFixed(4)}<br>
                            <strong>ID:</strong> ${id || 'N/A'}
                        </div>
                    </div>
                `);

            marker.on('click', () => {
                document.getElementById('selectedInfo').innerHTML = `
                    <h4>🧊 ${name}</h4>
                    <p><strong>Type:</strong> ${type}</p>
                    <p><strong>ID:</strong> ${id || 'N/A'}</p>
                    <p><strong>Latitude:</strong> ${lat.toFixed(6)}</p>
                    <p><strong>Longitude:</strong> ${lon.toFixed(6)}</p>
                `;
            });

            glacierMarkers.push(marker);
            validGlaciers++;

        } catch (error) {
            console.error(`❌ Error adding glacier ${name}:`, error);
            invalidGlaciers++;
        }
    });

    console.log(`📊 Glaciers summary: ${validGlaciers} valid, ${invalidGlaciers} invalid`);
}

function isValidCoordinate(lat, lon) {
    const isValid = (
        typeof lat === 'number' &&
        typeof lon === 'number' &&
        !isNaN(lat) &&
        !isNaN(lon) &&
        lat >= -90 && lat <= 90 &&
        lon >= -180 && lon <= 180
    );

    if (!isValid) {
        console.log('🔍 Coordinate validation failed:', { lat, lon, types: [typeof lat, typeof lon] });
    }

    return isValid;
}

function clearMarkers() {
    shipMarkers.forEach(marker => {
        try {
            map.removeLayer(marker);
        } catch (error) {
            console.warn('Error removing ship marker:', error);
        }
    });

    glacierMarkers.forEach(marker => {
        try {
            map.removeLayer(marker);
        } catch (error) {
            console.warn('Error removing glacier marker:', error);
        }
    });

    shipMarkers = [];
    glacierMarkers = [];

    document.getElementById('selectedInfo').innerHTML = '<p>Click on a ship or glacier for details</p>';

    console.log('🗑️ All markers cleared');
}

document.getElementById('loadData').addEventListener('click', loadData);

window.addEventListener('error', function(e) {
    console.error('Global error:', e.error);
});

map.setView([75.0, 40.0], 4);

console.log('🗺️ Map initialized');
console.log('📍 API Base:', API_BASE);