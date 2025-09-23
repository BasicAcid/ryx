// Ryx External Dashboard JavaScript
class RyxDashboard {
    constructor() {
        this.nodes = new Map();
        this.lastUpdateTime = Date.now();
        this.messageRate = 0;
        this.connectionStatus = 'connecting';
        this.updateInterval = null;
        this.dashboardBaseUrl = this.detectDashboardUrl();
        
        this.init();
    }

    detectDashboardUrl() {
        // Use the current dashboard location
        return `${window.location.protocol}//${window.location.host}`;
    }

    async init() {
        console.log('Initializing Ryx Dashboard...');
        this.setupEventListeners();
        this.setupTabs();
        await this.discoverCluster();
        this.startUpdateLoop();
    }

    setupEventListeners() {
        // Task submission form
        document.getElementById('task-form').addEventListener('submit', (e) => {
            e.preventDefault();
            this.submitTask();
        });

        // Topology controls
        document.getElementById('layout-button')?.addEventListener('click', () => {
            this.updateTopologyLayout();
        });

        document.getElementById('zoom-fit')?.addEventListener('click', () => {
            this.fitTopologyToScreen();
        });

        // Chemistry controls
        document.getElementById('inject-chemical')?.addEventListener('click', () => {
            this.injectChemical();
        });
    }

    setupTabs() {
        const tabButtons = document.querySelectorAll('.tab-button');
        const tabContents = document.querySelectorAll('.tab-content');

        tabButtons.forEach(button => {
            button.addEventListener('click', () => {
                const tabId = button.getAttribute('data-tab');
                
                // Remove active class from all tabs and contents
                tabButtons.forEach(b => b.classList.remove('active'));
                tabContents.forEach(c => c.classList.remove('active'));
                
                // Add active class to clicked tab and corresponding content
                button.classList.add('active');
                document.getElementById(`${tabId}-tab`).classList.add('active');
                
                // Trigger refresh for the active tab
                switch(tabId) {
                    case 'topology':
                        this.updateTopologyView();
                        break;
                    case 'chemistry':
                        this.updateChemistryView();
                        break;
                }
            });
        });
    }

    async discoverCluster() {
        console.log('Discovering cluster nodes...');
        this.updateConnectionStatus('connecting');
        
        try {
            // Use dashboard's cluster discovery API
            const clusterStatus = await this.fetchClusterStatus();
            
            if (clusterStatus && clusterStatus.nodes) {
                this.nodes.clear();
                
                for (const nodeInfo of clusterStatus.nodes) {
                    if (nodeInfo.reachable) {
                        this.nodes.set(nodeInfo.http_port, {
                            httpPort: nodeInfo.http_port,
                            port: nodeInfo.port,
                            id: nodeInfo.id || `node-${nodeInfo.http_port}`,
                            status: nodeInfo.reachable ? 'healthy' : 'error',
                            lastSeen: Date.now(),
                            data: nodeInfo
                        });
                    }
                }
            }
            
            this.updateConnectionStatus('connected');
            console.log(`Discovered ${this.nodes.size} nodes`);
        } catch (error) {
            console.error('Failed to discover cluster:', error);
            this.updateConnectionStatus('error');
        }
    }

    async fetchClusterStatus() {
        try {
            const response = await fetch(`${this.dashboardBaseUrl}/cluster/status`);
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }
            return await response.json();
        } catch (error) {
            console.error('Failed to fetch cluster status:', error);
            return null;
        }
    }

    async fetchFromNode(httpPort, endpoint) {
        try {
            const response = await fetch(`${this.dashboardBaseUrl}/node-proxy?port=${httpPort}&path=${encodeURIComponent(endpoint)}`);
            
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }
            
            return await response.json();
        } catch (error) {
            console.warn(`Failed to fetch ${endpoint} from node ${httpPort}:`, error);
            return null;
        }
    }

    async postToNode(httpPort, endpoint, data) {
        try {
            const response = await fetch(`${this.dashboardBaseUrl}/node-proxy?port=${httpPort}&path=${encodeURIComponent(endpoint)}`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(data)
            });
            
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }
            
            return await response.json();
        } catch (error) {
            console.warn(`Failed to post to ${endpoint} on node ${httpPort}:`, error);
            return null;
        }
    }

    startUpdateLoop() {
        this.updateInterval = setInterval(async () => {
            await this.updateAllNodes();
            this.updateUI();
        }, 2000); // Update every 2 seconds
    }

    async updateAllNodes() {
        // Use cluster status API to get fresh data
        const clusterStatus = await this.fetchClusterStatus();
        
        if (clusterStatus && clusterStatus.nodes) {
            // Update existing nodes or add new ones
            for (const nodeInfo of clusterStatus.nodes) {
                const existingNode = this.nodes.get(nodeInfo.http_port);
                
                if (existingNode) {
                    // Update existing node
                    existingNode.data = nodeInfo;
                    existingNode.status = nodeInfo.reachable ? 'healthy' : 'error';
                    existingNode.lastSeen = Date.now();
                } else if (nodeInfo.reachable) {
                    // Add new node
                    this.nodes.set(nodeInfo.http_port, {
                        httpPort: nodeInfo.http_port,
                        port: nodeInfo.port,
                        id: nodeInfo.id || `node-${nodeInfo.http_port}`,
                        status: 'healthy',
                        lastSeen: Date.now(),
                        data: nodeInfo
                    });
                }
            }
            
            // Remove nodes that are no longer in the cluster
            const activeHttpPorts = new Set(clusterStatus.nodes.map(n => n.http_port));
            for (const httpPort of this.nodes.keys()) {
                if (!activeHttpPorts.has(httpPort)) {
                    this.nodes.delete(httpPort);
                }
            }
        }
    }

    updateUI() {
        this.updateClusterStats();
        this.updateNodeGrid();
        this.enhanceNodeCards();
        
        // Update active tab content
        const activeTab = document.querySelector('.tab-button.active');
        if (activeTab) {
            const tabId = activeTab.getAttribute('data-tab');
            switch(tabId) {
                case 'topology':
                    this.updateTopologyView();
                    break;
                case 'chemistry':
                    this.updateChemistryView();
                    break;
            }
        }
    }

    updateConnectionStatus(status) {
        this.connectionStatus = status;
        const statusEl = document.getElementById('connection-status');
        const infoEl = document.getElementById('cluster-info');
        
        statusEl.className = `status-${status}`;
        
        switch(status) {
            case 'connecting':
                infoEl.textContent = 'Connecting to cluster...';
                break;
            case 'connected':
                infoEl.textContent = `Connected to ${this.nodes.size} nodes`;
                break;
            case 'error':
                infoEl.textContent = 'Connection error';
                break;
        }
    }

    updateClusterStats() {
        const totalNodes = this.nodes.size;
        const healthyNodes = Array.from(this.nodes.values()).filter(n => n.status === 'healthy').length;
        
        // Calculate message rate (simplified)
        const currentTime = Date.now();
        const timeDelta = (currentTime - this.lastUpdateTime) / 1000;
        this.lastUpdateTime = currentTime;
        
        document.getElementById('total-nodes').textContent = totalNodes;
        document.getElementById('healthy-nodes').textContent = healthyNodes;
        document.getElementById('active-tasks').textContent = '0'; // TODO: Calculate from all nodes
        document.getElementById('message-rate').textContent = Math.round(this.messageRate);
    }

    updateNodeGrid() {
        const grid = document.getElementById('node-grid');
        grid.innerHTML = '';

        if (this.nodes.size === 0) {
            grid.innerHTML = '<div class="loading">No nodes discovered</div>';
            return;
        }

        this.nodes.forEach((node, httpPort) => {
            const card = this.createNodeCard(node);
            grid.appendChild(card);
        });
    }

    createNodeCard(node) {
        const card = document.createElement('div');
        card.className = 'node-card';
        card.setAttribute('data-node-id', node.id);
        
        const statusClass = node.status === 'healthy' ? 'healthy' : 
                          node.status === 'warning' ? 'warning' : 
                          node.status === 'error' ? 'error' : 'offline';

        const coordinates = this.getNodeCoordinates(node.data);
        const zone = node.data.spatial?.zone || node.data.zone || 'unknown';
        const activeTasks = node.data.tasks || 0;
        const completedTasks = node.data.completed_tasks || 0;
        
        card.innerHTML = `
            <div class="node-card-header">
                <span class="node-id">${node.id}</span>
                <span class="node-status node-indicator ${statusClass}"></span>
            </div>
            <div class="node-info">
                <div><strong>Zone:</strong> ${zone}</div>
                <div><strong>Neighbors:</strong> ${node.data.neighbor_count || 0}</div>
                <div><strong>Active Tasks:</strong> <span class="active-tasks">${activeTasks}</span></div>
                <div><strong>Completed:</strong> <span class="completed-tasks">${completedTasks}</span></div>
                <div><strong>Status:</strong> ${node.data.last_update || 'unknown'}</div>
            </div>
        `;

        card.addEventListener('click', () => {
            this.showNodeDetails(node);
        });

        return card;
    }

    getNodeCoordinates(nodeData) {
        if (!nodeData.spatial) return 'none';
        
        const spatial = nodeData.spatial;
        const system = spatial.coordinate_system || 'none';
        
        if (system === 'none') return 'none';
        if (system === 'logical') return `zone: ${spatial.zone}`;
        
        const x = spatial.x !== undefined ? spatial.x.toFixed(2) : '?';
        const y = spatial.y !== undefined ? spatial.y.toFixed(2) : '?';
        const z = spatial.z !== undefined ? `, ${spatial.z.toFixed(2)}` : '';
        
        return `(${x}, ${y}${z})`;
    }

    formatUptime(seconds) {
        if (seconds < 60) return `${seconds}s`;
        if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
        if (seconds < 86400) return `${Math.floor(seconds / 3600)}h`;
        return `${Math.floor(seconds / 86400)}d`;
    }

    async submitTask() {
        const form = document.getElementById('task-form');
        const formData = new FormData(form);
        
        const taskData = {
            type: formData.get('type'),
            data: formData.get('data'),
            energy: parseFloat(formData.get('energy')),
            ttl: 3600 // 1 hour
        };

        console.log('Submitting task:', taskData);

        // Submit to the first available healthy node
        const healthyNode = Array.from(this.nodes.values()).find(n => n.status === 'healthy');
        if (!healthyNode) {
            alert('No healthy nodes available');
            return;
        }

        try {
            // Use cluster-wide task submission
            const result = await this.submitTaskToCluster(taskData);
            if (result && result.success) {
                console.log('Task submitted successfully:', result);
                this.addTaskToRecentList(taskData, result);
                this.animateTaskSubmission(result.task.id);
                form.reset();
                document.getElementById('task-energy').value = '10'; // Reset to default
            } else {
                alert('Failed to submit task');
            }
        } catch (error) {
            console.error('Error submitting task:', error);
            alert('Error submitting task: ' + error.message);
        }
    }

    addTaskToRecentList(taskData, result) {
        const taskList = document.getElementById('recent-tasks');
        const noTasksEl = taskList.querySelector('.no-tasks');
        if (noTasksEl) {
            noTasksEl.remove();
        }

        const taskItem = document.createElement('div');
        taskItem.className = 'task-item';
        
        const taskId = result.task_id || 'unknown';
        
        taskItem.innerHTML = `
            <div class="task-item-header">
                <span class="task-id">${taskId.substring(0, 8)}...</span>
                <span class="task-status active">Active</span>
            </div>
            <div class="task-data">${taskData.type}: "${taskData.data.substring(0, 30)}${taskData.data.length > 30 ? '...' : ''}"</div>
        `;

        taskList.insertBefore(taskItem, taskList.firstChild);

        // Remove old tasks if we have too many
        const taskItems = taskList.querySelectorAll('.task-item');
        if (taskItems.length > 10) {
            taskItems[taskItems.length - 1].remove();
        }
    }

    updateTopologyView() {
        // Simple SVG-based topology view
        const svg = document.getElementById('topology-svg');
        svg.innerHTML = '';

        if (this.nodes.size === 0) return;

        const width = svg.clientWidth || 800;
        const height = svg.clientHeight || 500;
        
        // Position nodes in a circle
        const centerX = width / 2;
        const centerY = height / 2;
        const radius = Math.min(width, height) * 0.3;
        
        const nodeArray = Array.from(this.nodes.values());
        nodeArray.forEach((node, index) => {
            const angle = (index / nodeArray.length) * 2 * Math.PI;
            const x = centerX + radius * Math.cos(angle);
            const y = centerY + radius * Math.sin(angle);
            
            // Draw node
            const circle = document.createElementNS('http://www.w3.org/2000/svg', 'circle');
            circle.setAttribute('cx', x);
            circle.setAttribute('cy', y);
            circle.setAttribute('r', 15);
            circle.setAttribute('fill', node.status === 'healthy' ? '#4caf50' : '#f44336');
            circle.setAttribute('stroke', '#4fc3f7');
            circle.setAttribute('stroke-width', '2');
            
            // Add node label
            const text = document.createElementNS('http://www.w3.org/2000/svg', 'text');
            text.setAttribute('x', x);
            text.setAttribute('y', y + 30);
            text.setAttribute('text-anchor', 'middle');
            text.setAttribute('fill', '#e6e6e6');
            text.setAttribute('font-size', '12');
            text.textContent = node.id.substring(0, 8);
            
            svg.appendChild(circle);
            svg.appendChild(text);
        });

        // Draw connections (simplified - connect all nodes for now)
        for (let i = 0; i < nodeArray.length; i++) {
            for (let j = i + 1; j < nodeArray.length; j++) {
                const angle1 = (i / nodeArray.length) * 2 * Math.PI;
                const angle2 = (j / nodeArray.length) * 2 * Math.PI;
                
                const x1 = centerX + radius * Math.cos(angle1);
                const y1 = centerY + radius * Math.sin(angle1);
                const x2 = centerX + radius * Math.cos(angle2);
                const y2 = centerY + radius * Math.sin(angle2);
                
                const line = document.createElementNS('http://www.w3.org/2000/svg', 'line');
                line.setAttribute('x1', x1);
                line.setAttribute('y1', y1);
                line.setAttribute('x2', x2);
                line.setAttribute('y2', y2);
                line.setAttribute('stroke', '#0f3460');
                line.setAttribute('stroke-width', '1');
                line.setAttribute('opacity', '0.5');
                
                svg.insertBefore(line, svg.firstChild); // Insert behind nodes
            }
        }
    }

    updateTopologyLayout() {
        this.updateTopologyView();
    }

    fitTopologyToScreen() {
        this.updateTopologyView();
    }

    async updateChemistryView() {
        const grid = document.getElementById('chemistry-grid');
        
        // Get chemistry data from first available node
        const healthyNode = Array.from(this.nodes.values()).find(n => n.status === 'healthy');
        if (!healthyNode) {
            grid.innerHTML = '<div class="loading">No healthy nodes available</div>';
            return;
        }

        try {
            const concentrations = await this.fetchFromNode(healthyNode.httpPort, '/chemistry/concentrations');
            const stats = await this.fetchFromNode(healthyNode.httpPort, '/chemistry/stats');
            
            if (concentrations && stats) {
                this.updateChemistryStats(stats);
                this.renderChemistryGrid(concentrations);
            } else {
                grid.innerHTML = '<div class="loading">No chemistry data available</div>';
            }
        } catch (error) {
            console.error('Error updating chemistry view:', error);
            grid.innerHTML = '<div class="loading">Error loading chemistry data</div>';
        }
    }

    updateChemistryStats(stats) {
        document.getElementById('total-concentration').textContent = 
            (stats.total_concentration || 0).toFixed(2);
        document.getElementById('active-reactions').textContent = 
            stats.total_reactions || 0;
    }

    renderChemistryGrid(concentrations) {
        const grid = document.getElementById('chemistry-grid');
        grid.innerHTML = '';

        if (!concentrations.concentrations || Object.keys(concentrations.concentrations).length === 0) {
            grid.innerHTML = '<div class="loading">No chemical concentrations detected</div>';
            return;
        }

        Object.entries(concentrations.concentrations).forEach(([type, concentration]) => {
            const card = document.createElement('div');
            card.className = 'chemical-card';
            
            const percentage = Math.min(concentration * 100, 100);
            
            card.innerHTML = `
                <div class="chemical-type">${type}</div>
                <div class="concentration-bar">
                    <div class="concentration-fill" style="width: ${percentage}%"></div>
                </div>
                <div style="font-size: 12px; color: #b0b0b0;">
                    Concentration: ${concentration.toFixed(4)}
                </div>
            `;
            
            grid.appendChild(card);
        });
    }

    async injectChemical() {
        const healthyNode = Array.from(this.nodes.values()).find(n => n.status === 'healthy');
        if (!healthyNode) {
            alert('No healthy nodes available');
            return;
        }

        const chemicalData = {
            type: 'test_chemical',
            content: 'Test chemical injection from dashboard',
            energy: 20.0,
            ttl: 300
        };

        try {
            const result = await this.postToNode(healthyNode.httpPort, '/inject', chemicalData);
            if (result) {
                console.log('Chemical injected successfully:', result);
                // Refresh chemistry view
                setTimeout(() => this.updateChemistryView(), 1000);
            } else {
                alert('Failed to inject chemical');
            }
        } catch (error) {
            console.error('Error injecting chemical:', error);
            alert('Error injecting chemical: ' + error.message);
        }
    }

    showNodeDetails(node) {
        // Simple alert for now - could be expanded to a modal
        const details = `
Node ID: ${node.id}
Status: ${node.status}
HTTP Port: ${node.httpPort}
UDP Port: ${node.port}
Zone: ${node.data.zone || 'unknown'}
Neighbors: ${node.data.neighbors || 0}
Tasks: ${node.data.tasks || 0}
Reachable: ${node.data.reachable ? 'Yes' : 'No'}
        `;
        alert(details);
    }

    async submitTaskToCluster(taskData) {
        try {
            const response = await fetch(`${this.dashboardBaseUrl}/cluster/submit-task`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(taskData)
            });
            
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            
            return await response.json();
        } catch (error) {
            console.error('Cluster task submission failed:', error);
            throw error;
        }
    }

    animateTaskSubmission(taskId) {
        // Add visual indicator to show task was submitted
        const indicator = document.createElement('div');
        indicator.className = 'task-submission-indicator';
        indicator.textContent = `Task ${taskId.substring(0,8)} submitted`;
        indicator.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            background: #4CAF50;
            color: white;
            padding: 10px 15px;
            border-radius: 5px;
            z-index: 1000;
            opacity: 1;
            transition: all 0.5s ease;
            font-weight: bold;
        `;
        
        document.body.appendChild(indicator);
        
        // Animate and remove
        setTimeout(() => {
            indicator.style.opacity = '0';
            indicator.style.transform = 'translateY(-20px)';
        }, 2000);
        
        setTimeout(() => {
            document.body.removeChild(indicator);
        }, 2500);
    }

    addTaskFlowIndicator(nodeId, taskId) {
        // Add flowing indicator on specific node
        const nodeCard = document.querySelector(`[data-node-id="${nodeId}"]`);
        if (!nodeCard) return;
        
        const flowIndicator = document.createElement('div');
        flowIndicator.className = 'task-flow-indicator';
        flowIndicator.textContent = '⚡';
        flowIndicator.style.cssText = `
            position: absolute;
            top: 5px;
            right: 5px;
            background: #FF9800;
            color: white;
            width: 20px;
            height: 20px;
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 12px;
            animation: pulse 1s infinite;
        `;
        
        nodeCard.style.position = 'relative';
        nodeCard.appendChild(flowIndicator);
        
        // Remove after 3 seconds
        setTimeout(() => {
            if (flowIndicator.parentNode) {
                flowIndicator.parentNode.removeChild(flowIndicator);
            }
        }, 3000);
    }

    enhanceNodeCards() {
        // Add task completion animations to node cards
        this.nodes.forEach((node, httpPort) => {
            const nodeCard = document.querySelector(`[data-node-id="${node.id}"]`);
            if (nodeCard) {
                // Add CSS classes for animation
                if (node.data.tasks > 0) {
                    nodeCard.classList.add('node-active');
                } else {
                    nodeCard.classList.remove('node-active');
                }
                
                // Update completion count with animation
                const completedEl = nodeCard.querySelector('.completed-tasks');
                if (completedEl) {
                    const currentCount = parseInt(completedEl.textContent) || 0;
                    const newCount = node.data.completed_tasks || 0;
                    
                    if (newCount > currentCount) {
                        // Animate task completion
                        completedEl.style.transform = 'scale(1.2)';
                        completedEl.style.color = '#4CAF50';
                        setTimeout(() => {
                            completedEl.style.transform = 'scale(1)';
                            completedEl.style.color = '';
                        }, 300);
                    }
                    
                    completedEl.textContent = newCount;
                }
            }
        });
    }

    destroy() {
        if (this.updateInterval) {
            clearInterval(this.updateInterval);
        }
    }
}

// Initialize dashboard when page loads
document.addEventListener('DOMContentLoaded', () => {
    window.ryxDashboard = new RyxDashboard();
});

// Clean up on page unload
window.addEventListener('beforeunload', () => {
    if (window.ryxDashboard) {
        window.ryxDashboard.destroy();
    }
});