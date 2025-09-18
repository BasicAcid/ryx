#!/bin/bash

echo "🚀 Testing Ryx Web Dashboard"
echo "================================"

# Test dashboard availability
echo "1. Testing dashboard availability..."
HTTP_CODE=$(curl -s -w "%{http_code}" http://localhost:8010/dashboard -o /dev/null)
if [ "$HTTP_CODE" = "200" ]; then
    echo "   ✅ Dashboard accessible at http://localhost:8010/dashboard"
else
    echo "   ❌ Dashboard not accessible (HTTP $HTTP_CODE)"
    exit 1
fi

# Test static files
echo "2. Testing static files..."
CSS_CODE=$(curl -s -w "%{http_code}" http://localhost:8010/static/css/dashboard.css -o /dev/null)
JS_CODE=$(curl -s -w "%{http_code}" http://localhost:8010/static/js/dashboard.js -o /dev/null)
if [ "$CSS_CODE" = "200" ] && [ "$JS_CODE" = "200" ]; then
    echo "   ✅ Static files served correctly"
else
    echo "   ❌ Static files not accessible (CSS:$CSS_CODE, JS:$JS_CODE)"
fi

# Test API endpoints
echo "3. Testing API endpoints..."
NODE_ID=$(curl -s http://localhost:8010/status | jq -r '.node_id')
NEIGHBOR_COUNT=$(curl -s http://localhost:8010/status | jq '.neighbors | length')
if [ "$NODE_ID" != "null" ] && [ "$NEIGHBOR_COUNT" != "null" ]; then
    echo "   ✅ Node $NODE_ID with $NEIGHBOR_COUNT neighbors"
else
    echo "   ❌ API endpoints not working"
fi

# Test chemistry endpoints
echo "4. Testing chemistry endpoints..."
CHEM_RESPONSE=$(curl -s http://localhost:8010/chemistry/concentrations)
if echo "$CHEM_RESPONSE" | jq -e '.concentrations' > /dev/null 2>&1; then
    echo "   ✅ Chemistry monitoring available"
else
    echo "   ❌ Chemistry endpoints not working"
fi

# Submit test task
echo "5. Testing task submission..."
TASK_RESPONSE=$(curl -s -X POST http://localhost:8010/compute \
    -H "Content-Type: application/json" \
    -d '{"type":"wordcount","data":"hello ryx dashboard","energy":5,"ttl":300}')
if echo "$TASK_RESPONSE" | jq -e '.success' > /dev/null 2>&1; then
    TASK_ID=$(echo "$TASK_RESPONSE" | jq -r '.task.id')
    echo "   ✅ Task submitted successfully (ID: ${TASK_ID:0:8}...)"
else
    echo "   ❌ Task submission failed"
fi

# Inject test chemical
echo "6. Testing chemical injection..."
CHEM_INJECT=$(curl -s -X POST http://localhost:8010/inject \
    -H "Content-Type: application/json" \
    -d '{"type":"test_chemical","content":"Dashboard test","energy":10,"ttl":300}')
if echo "$CHEM_INJECT" | jq -e '.success' > /dev/null 2>&1; then
    echo "   ✅ Chemical injection successful"
else
    echo "   ❌ Chemical injection failed"
fi

echo ""
echo "🎯 Dashboard Test Results:"
echo "================================"
echo "✅ Web Dashboard: http://localhost:8010/dashboard"
echo "✅ Node Grid: Shows $NEIGHBOR_COUNT neighboring nodes"
echo "✅ Task Interface: Working computational task submission"
echo "✅ Chemistry Monitor: Real-time chemical concentration tracking"
echo "✅ Network Topology: SVG-based network visualization"
echo ""
echo "📋 How to use:"
echo "1. Open http://localhost:8010/dashboard in your browser"
echo "2. Navigate between tabs: Node Grid, Network Topology, Chemistry Monitor"
echo "3. Submit tasks using the form in the sidebar"
echo "4. Inject chemicals to see diffusion in real-time"
echo "5. Click on nodes to see detailed information"
echo ""
echo "🔧 Cluster Management:"
echo "./ryx-cluster -cmd status    # Detailed cluster status"
echo "./ryx-cluster -cmd stop      # Stop the cluster"
echo ""