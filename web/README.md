# Ryx Web Dashboard

A real-time web interface for observing and interacting with the Ryx distributed computing cluster.

## Features

### 🎯 Node Grid View
- **Live cluster visualization**: Real-time grid showing all cluster nodes
- **Health indicators**: Color-coded status (healthy, warning, error, offline)
- **Node details**: ID, zone, coordinates, neighbor count, message count, uptime
- **Spatial awareness**: Display coordinate systems and zone information

### 🌐 Network Topology View
- **Interactive topology graph**: SVG-based visualization of node connections
- **Real-time updates**: Live network structure changes
- **Force layout**: Automatic positioning of nodes for optimal visualization
- **Connection visualization**: Lines showing neighbor relationships

### ⚗️ Chemistry Monitor
- **Real-time concentrations**: Live chemical concentration tracking
- **Reaction monitoring**: Active chemical reactions and statistics
- **Chemical injection**: Test chemical diffusion through the cluster
- **Concentration visualization**: Bar charts showing chemical distribution

### 📊 Task Management
- **Task submission**: Simple forms for submitting computational tasks
- **Progress monitoring**: Real-time task status and completion tracking
- **Recent task history**: View recently submitted and completed tasks
- **Cluster statistics**: Live metrics for nodes, tasks, and system health

## Getting Started

### 1. Start a Ryx Cluster

```bash
# Build the binaries
go build -o ryx-node ./cmd/ryx-node
go build -o ryx-cluster ./cmd/ryx-cluster

# Start a test cluster
./ryx-cluster -cmd start -profile small
```

### 2. Access the Dashboard

Open your browser and navigate to any cluster node:
- **Primary node**: http://localhost:8010/
- **Other nodes**: http://localhost:8011/, http://localhost:8012/, etc.

### 3. Explore the Interface

**Node Grid Tab**:
- View all discovered cluster nodes
- Click on nodes to see detailed information
- Observe health status changes in real-time

**Network Topology Tab**:
- See visual representation of node connections
- Use "Force Layout" and "Fit to Screen" controls
- Watch topology changes as nodes join/leave

**Chemistry Monitor Tab**:
- Monitor chemical concentrations across the cluster
- Click "Inject Chemical" to test diffusion
- Observe concentration changes in real-time

**Task Submission**:
- Select task type (currently supports "Word Count")
- Enter input data and energy level
- Submit tasks and watch them execute across the cluster

## Advanced Usage

### Spatial Clusters

For spatial awareness features, start nodes with coordinates:

```bash
# GPS coordinates (smart city scenario)
./ryx-node --coord-system gps --x 40.7128 --y -74.0060 --zone datacenter_a --port 9010 --http-port 8010

# Relative coordinates (vehicle scenario)  
./ryx-node --coord-system relative --x 15.2 --y -3.1 --z 2.8 --zone bridge --port 9011 --http-port 8011
```

### Chemistry Testing

```bash
# Inject chemical messages
curl -X POST http://localhost:8010/inject -H "Content-Type: application/json" \
  -d '{"type": "oxygen", "content": "O2", "energy": 20.0, "ttl": 300}'

# Monitor concentrations
curl -s http://localhost:8010/chemistry/concentrations | jq '.'
```

### Task Submission via API

```bash
# Submit computational task
curl -X POST http://localhost:8010/compute -H "Content-Type: application/json" \
  -d '{"type":"wordcount","data":"distributed computing test","energy":15,"ttl":3600}'

# Check results
curl -s http://localhost:8010/compute | jq '.'
```

## Technical Details

### Architecture
- **Frontend**: Pure HTML/CSS/JavaScript (no external dependencies)
- **Backend Integration**: Uses existing Ryx HTTP API endpoints
- **Real-time Updates**: JavaScript polling (2-second intervals)
- **Static Files**: CSS and JS served by Ryx HTTP server

### Browser Compatibility
- Modern browsers with JavaScript enabled
- Responsive design works on desktop and mobile
- SVG support required for topology visualization

### Performance
- **Lightweight**: Minimal network overhead from polling
- **Scalable**: Works with clusters of 5-50+ nodes
- **Efficient**: Only fetches changed data
- **Responsive**: 2-second update intervals provide near real-time experience

## Troubleshooting

### Dashboard Not Loading
1. Ensure Ryx node is running: `ps aux | grep ryx-node`
2. Check HTTP port is accessible: `curl http://localhost:8010/status`
3. Verify static files exist: `ls web/static/`

### No Nodes Appearing
1. Check cluster startup: `./ryx-cluster -cmd status`
2. Verify discovery is working: `curl http://localhost:8010/status | jq '.neighbors'`
3. Ensure nodes are healthy: All should return 200 on `/status`

### Chemistry Data Missing
1. Inject test chemicals: Use "Inject Chemical" button or API
2. Check chemistry endpoints: `curl http://localhost:8010/chemistry/concentrations`
3. Verify chemistry service is running: Check node logs

### Task Submission Failing
1. Verify computation service: `curl http://localhost:8010/compute`
2. Check task format: Ensure JSON is valid
3. Monitor node logs for error messages

## Development

### Adding New Features
- **CSS**: Edit `web/static/css/dashboard.css`
- **JavaScript**: Edit `web/static/js/dashboard.js`
- **HTML**: Edit the embedded HTML in `internal/api/server.go`

### WebSocket Integration (Future)
The current implementation uses HTTP polling. WebSocket support can be added for true real-time updates with lower latency and network overhead.

## Future Enhancements

- **WebSocket real-time updates**: Reduce latency and network usage
- **Historical data**: Charts showing performance trends over time
- **Advanced topology layouts**: Multiple visualization algorithms
- **Task templates**: Pre-configured task types for common operations
- **Cluster configuration**: Runtime parameter modification via UI
- **Alert system**: Visual and audio notifications for failures
- **Export capabilities**: Download results and metrics data