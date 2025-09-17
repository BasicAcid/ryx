package chemistry

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/BasicAcid/ryx/internal/types"
)

// Engine implements chemistry-based computing for message reactions and concentration tracking
// Phase 4A: Core Chemistry Foundation
type Engine struct {
	nodeID             string
	concentrationState *types.ConcentrationState
	reactionHistory    []*types.ChemicalReaction
	mu                 sync.RWMutex

	// Configuration
	maxReactionHistory int
	concentrationDecay float64 // Rate at which concentrations decay over time
	diffusionThreshold float64 // Minimum concentration difference for diffusion
	reactionThreshold  float64 // Minimum energy for reactions to occur

	// Chemical parameters
	baseReactionRate   float64 // Base probability for chemical reactions
	catalystBoost      float64 // Reaction rate multiplier for catalysts
	inhibitorReduction float64 // Reaction rate reduction for inhibitors
}

// NewEngine creates a new chemistry engine for a node
func NewEngine(nodeID string) *Engine {
	return &Engine{
		nodeID: nodeID,
		concentrationState: &types.ConcentrationState{
			MessageCounts:   make(map[string]int),
			TotalMessages:   0,
			Concentrations:  make(map[string]float64),
			GradientVectors: make(map[string]float64),
			LastUpdate:      time.Now().Unix(),
		},
		reactionHistory:    make([]*types.ChemicalReaction, 0),
		maxReactionHistory: 100,
		concentrationDecay: 0.01, // 1% decay per update
		diffusionThreshold: 0.05, // 5% concentration difference for diffusion
		reactionThreshold:  0.1,  // Minimum 0.1 energy for reactions
		baseReactionRate:   0.1,  // 10% base reaction probability
		catalystBoost:      2.0,  // 2x reaction rate with catalyst
		inhibitorReduction: 0.5,  // 50% reaction rate reduction with inhibitor
	}
}

// UpdateConcentrations updates chemical concentrations based on current messages
func (e *Engine) UpdateConcentrations(messages []*types.InfoMessage) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Reset counts
	e.concentrationState.MessageCounts = make(map[string]int)
	e.concentrationState.TotalMessages = len(messages)

	// Count messages by type
	for _, msg := range messages {
		e.concentrationState.MessageCounts[msg.Type]++
	}

	// Calculate concentrations (normalized by total messages)
	e.concentrationState.Concentrations = make(map[string]float64)
	if e.concentrationState.TotalMessages > 0 {
		for msgType, count := range e.concentrationState.MessageCounts {
			concentration := float64(count) / float64(e.concentrationState.TotalMessages)
			e.concentrationState.Concentrations[msgType] = concentration
		}
	}

	// Apply concentration decay over time
	e.applyConcentrationDecay()

	e.concentrationState.LastUpdate = time.Now().Unix()

	log.Printf("Chemistry[%s]: Updated concentrations - total messages: %d, types: %d",
		e.nodeID, e.concentrationState.TotalMessages, len(e.concentrationState.Concentrations))
}

// ProcessChemicalReactions attempts to perform chemical reactions between messages
func (e *Engine) ProcessChemicalReactions(messages []*types.InfoMessage) ([]*types.InfoMessage, []*types.ChemicalReaction) {
	e.mu.Lock()
	defer e.mu.Unlock()

	var newMessages []*types.InfoMessage
	var reactions []*types.ChemicalReaction

	// Process potential reactions between messages
	for i, msg1 := range messages {
		if msg1.Chemical == nil {
			continue
		}

		for j, msg2 := range messages {
			if i >= j || msg2.Chemical == nil {
				continue
			}

			// Attempt reaction between msg1 and msg2
			if reaction := e.attemptReaction(msg1, msg2); reaction != nil {
				// Create product message
				product := e.createProductMessage(msg1, msg2, reaction)
				if product != nil {
					newMessages = append(newMessages, product)
					reactions = append(reactions, reaction)

					log.Printf("Chemistry[%s]: Reaction %s - %s + %s → %s (energy: %.2f)",
						e.nodeID, reaction.ReactionType, msg1.Type, msg2.Type, product.Type, reaction.EnergyChange)
				}
			}
		}

		// Check for catalytic reactions (message catalyzes without being consumed)
		if msg1.Chemical.Catalyst {
			catalyzedProducts := e.processCatalyticReactions(msg1, messages)
			newMessages = append(newMessages, catalyzedProducts...)
		}
	}

	// Store reactions in history
	e.reactionHistory = append(e.reactionHistory, reactions...)

	// Trim reaction history if needed
	if len(e.reactionHistory) > e.maxReactionHistory {
		e.reactionHistory = e.reactionHistory[len(e.reactionHistory)-e.maxReactionHistory:]
	}

	return newMessages, reactions
}

// CalculateEnergyDecay calculates energy decay based on distance and chemical properties
func (e *Engine) CalculateEnergyDecay(msg *types.InfoMessage, distance float64, baseDecayRate float64) float64 {
	// Base energy decay
	decay := baseDecayRate

	// Chemical modifications to decay rate
	if msg.Chemical != nil {
		// Reactive messages lose energy faster
		decay += msg.Chemical.Reactivity * 0.1

		// Catalyst messages retain energy better
		if msg.Chemical.Catalyst {
			decay *= 0.8
		}

		// Messages with high concentration diffuse more efficiently
		if concentration, exists := e.concentrationState.Concentrations[msg.Type]; exists {
			// Higher concentration = lower decay (chemical momentum)
			decay *= (1.0 - concentration*0.3)
		}
	}

	// Distance-based decay (spatial chemistry)
	if distance > 0 {
		// Energy consumption proportional to distance
		distanceDecay := distance * 0.01 // 1% per unit distance
		decay += distanceDecay
	}

	// Ensure reasonable bounds
	decay = math.Max(0.01, math.Min(0.5, decay))

	return decay
}

// GetConcentrationGradient calculates concentration gradient for diffusion
func (e *Engine) GetConcentrationGradient(msgType string, neighborConcentrations map[string]float64) float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()

	localConcentration, exists := e.concentrationState.Concentrations[msgType]
	if !exists {
		localConcentration = 0.0
	}

	// Calculate average neighbor concentration
	var totalConcentration float64
	var count int
	for _, concentration := range neighborConcentrations {
		totalConcentration += concentration
		count++
	}

	if count == 0 {
		return 0.0
	}

	avgNeighborConcentration := totalConcentration / float64(count)

	// Gradient is difference between neighbor average and local concentration
	gradient := avgNeighborConcentration - localConcentration

	// Only significant gradients trigger diffusion
	if math.Abs(gradient) < e.diffusionThreshold {
		return 0.0
	}

	return gradient
}

// attemptReaction tries to create a chemical reaction between two messages
func (e *Engine) attemptReaction(msg1, msg2 *types.InfoMessage) *types.ChemicalReaction {
	// Check if either message has reaction rules
	var rules []*types.ReactionRule
	var primary, secondary *types.InfoMessage

	if len(msg1.Chemical.ReactionRules) > 0 {
		rules = msg1.Chemical.ReactionRules
		primary = msg1
		secondary = msg2
	} else if len(msg2.Chemical.ReactionRules) > 0 {
		rules = msg2.Chemical.ReactionRules
		primary = msg2
		secondary = msg1
	} else {
		return nil // No reaction rules
	}

	// Check each reaction rule
	for _, rule := range rules {
		if e.canReact(primary, secondary, rule) {
			// Calculate reaction probability
			probability := e.calculateReactionProbability(primary, secondary, rule)

			if rand.Float64() < probability {
				// Create reaction
				reaction := &types.ChemicalReaction{
					ReactionID:   e.generateReactionID(primary, secondary),
					ReactantIDs:  []string{primary.ID, secondary.ID},
					ProductID:    "", // Will be set when product is created
					ReactionType: rule.ReactionType,
					EnergyChange: rule.EnergyChange,
					NodeID:       e.nodeID,
					Timestamp:    time.Now().Unix(),
				}

				return reaction
			}
		}
	}

	return nil
}

// canReact checks if two messages can react according to a rule
func (e *Engine) canReact(primary, secondary *types.InfoMessage, rule *types.ReactionRule) bool {
	// Check energy requirement
	if primary.Energy < rule.RequiredEnergy {
		return false
	}

	// Check target type
	if rule.TargetType != "" && secondary.Type != rule.TargetType {
		return false
	}

	// Check affinity tags
	if len(rule.TargetTags) > 0 {
		if secondary.Chemical == nil || len(secondary.Chemical.AffinityTags) == 0 {
			return false
		}

		// Check if any target tags match affinity tags
		hasMatch := false
		for _, targetTag := range rule.TargetTags {
			for _, affinityTag := range secondary.Chemical.AffinityTags {
				if targetTag == affinityTag {
					hasMatch = true
					break
				}
			}
			if hasMatch {
				break
			}
		}

		if !hasMatch {
			return false
		}
	}

	return true
}

// calculateReactionProbability calculates the probability of a reaction occurring
func (e *Engine) calculateReactionProbability(primary, secondary *types.InfoMessage, rule *types.ReactionRule) float64 {
	probability := rule.Probability

	// Base reaction rate
	probability *= e.baseReactionRate

	// Reactivity influences probability
	if primary.Chemical != nil {
		probability *= (1.0 + primary.Chemical.Reactivity)
	}
	if secondary.Chemical != nil {
		probability *= (1.0 + secondary.Chemical.Reactivity)
	}

	// Catalyst effects
	if primary.Chemical != nil && primary.Chemical.Catalyst {
		probability *= e.catalystBoost
	}
	if secondary.Chemical != nil && secondary.Chemical.Catalyst {
		probability *= e.catalystBoost
	}

	// Inhibitor effects
	if primary.Chemical != nil && primary.Chemical.Inhibitor {
		probability *= e.inhibitorReduction
	}
	if secondary.Chemical != nil && secondary.Chemical.Inhibitor {
		probability *= e.inhibitorReduction
	}

	// Concentration effects (higher concentration = higher reaction rate)
	if primaryConcentration, exists := e.concentrationState.Concentrations[primary.Type]; exists {
		probability *= (1.0 + primaryConcentration)
	}

	return math.Min(1.0, probability)
}

// createProductMessage creates a new message from a chemical reaction
func (e *Engine) createProductMessage(msg1, msg2 *types.InfoMessage, reaction *types.ChemicalReaction) *types.InfoMessage {
	// Determine product type from reaction
	productType := "reaction_product"
	if reaction.ReactionType == "combine" {
		productType = fmt.Sprintf("%s_%s_combined", msg1.Type, msg2.Type)
	} else if reaction.ReactionType == "transform" {
		productType = fmt.Sprintf("%s_transformed", msg1.Type)
	}

	// Combine content (simplified)
	combinedContent := fmt.Sprintf("Reaction product from %s + %s", string(msg1.Content), string(msg2.Content))

	// Calculate product energy
	productEnergy := (msg1.Energy + msg2.Energy) + reaction.EnergyChange
	if productEnergy < 0 {
		productEnergy = 0.1 // Minimum energy
	}

	// Create product message
	product := &types.InfoMessage{
		ID:        e.generateMessageID(combinedContent),
		Type:      productType,
		Content:   []byte(combinedContent),
		Energy:    productEnergy,
		TTL:       time.Now().Add(time.Hour).Unix(), // 1 hour TTL
		Hops:      0,
		Source:    e.nodeID,
		Path:      []string{e.nodeID},
		Timestamp: time.Now().Unix(),
		Metadata:  make(map[string]interface{}),
		Chemical: &types.ChemicalProperties{
			Concentration:  0.1,
			Reactivity:     0.3,
			Catalyst:       false,
			Inhibitor:      false,
			DiffusionRate:  0.5,
			SourceStrength: 0.8,
			ChemicalType:   "product",
			AffinityTags:   []string{"reaction_product"},
		},
	}

	// Set product ID in reaction
	reaction.ProductID = product.ID

	return product
}

// processCatalyticReactions handles catalytic reactions that don't consume the catalyst
func (e *Engine) processCatalyticReactions(catalyst *types.InfoMessage, messages []*types.InfoMessage) []*types.InfoMessage {
	var products []*types.InfoMessage

	// Catalysts can accelerate reactions between other messages
	for i, msg1 := range messages {
		for j, msg2 := range messages {
			if i >= j || msg1.ID == catalyst.ID || msg2.ID == catalyst.ID {
				continue
			}

			// Check if catalyst can accelerate reaction between msg1 and msg2
			if e.canCatalyze(catalyst, msg1, msg2) {
				// Create catalyzed product with boosted probability
				if rand.Float64() < 0.3 { // 30% chance for catalytic reaction
					product := e.createCatalyzedProduct(catalyst, msg1, msg2)
					if product != nil {
						products = append(products, product)
					}
				}
			}
		}
	}

	return products
}

// canCatalyze checks if a catalyst can accelerate a reaction
func (e *Engine) canCatalyze(catalyst, msg1, msg2 *types.InfoMessage) bool {
	if catalyst.Chemical == nil || !catalyst.Chemical.Catalyst {
		return false
	}

	// Simple catalysis rules - catalyst must have affinity for one of the reactants
	if len(catalyst.Chemical.AffinityTags) == 0 {
		return false
	}

	// Check if catalyst has affinity for either reactant type
	for _, tag := range catalyst.Chemical.AffinityTags {
		if tag == msg1.Type || tag == msg2.Type {
			return true
		}
	}

	return false
}

// createCatalyzedProduct creates a product from catalytic reaction
func (e *Engine) createCatalyzedProduct(catalyst, msg1, msg2 *types.InfoMessage) *types.InfoMessage {
	// Create enhanced product due to catalysis
	combinedContent := fmt.Sprintf("Catalyzed reaction: %s + %s (catalyst: %s)",
		string(msg1.Content), string(msg2.Content), catalyst.Type)

	productEnergy := (msg1.Energy + msg2.Energy) * 1.2 // 20% energy boost from catalysis

	product := &types.InfoMessage{
		ID:        e.generateMessageID(combinedContent),
		Type:      fmt.Sprintf("%s_%s_catalyzed", msg1.Type, msg2.Type),
		Content:   []byte(combinedContent),
		Energy:    productEnergy,
		TTL:       time.Now().Add(2 * time.Hour).Unix(), // Longer TTL for catalyzed products
		Hops:      0,
		Source:    e.nodeID,
		Path:      []string{e.nodeID},
		Timestamp: time.Now().Unix(),
		Metadata:  make(map[string]interface{}),
		Chemical: &types.ChemicalProperties{
			Concentration:  0.2,
			Reactivity:     0.4,
			Catalyst:       false,
			Inhibitor:      false,
			DiffusionRate:  0.7,
			SourceStrength: 1.0,
			ChemicalType:   "catalyzed_product",
			AffinityTags:   []string{"catalyzed", "enhanced"},
		},
	}

	return product
}

// applyConcentrationDecay reduces concentrations over time
func (e *Engine) applyConcentrationDecay() {
	for msgType, concentration := range e.concentrationState.Concentrations {
		newConcentration := concentration * (1.0 - e.concentrationDecay)
		if newConcentration < 0.001 { // Remove very low concentrations
			delete(e.concentrationState.Concentrations, msgType)
		} else {
			e.concentrationState.Concentrations[msgType] = newConcentration
		}
	}
}

// Helper functions

func (e *Engine) generateReactionID(msg1, msg2 *types.InfoMessage) string {
	data := fmt.Sprintf("%s_%s_%d", msg1.ID, msg2.ID, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])[:16]
}

func (e *Engine) generateMessageID(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])[:16]
}

// Getters for status and monitoring

// GetConcentrationState returns the current concentration state
func (e *Engine) GetConcentrationState() *types.ConcentrationState {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Return a copy to avoid race conditions
	state := &types.ConcentrationState{
		MessageCounts:   make(map[string]int),
		TotalMessages:   e.concentrationState.TotalMessages,
		Concentrations:  make(map[string]float64),
		GradientVectors: make(map[string]float64),
		LastUpdate:      e.concentrationState.LastUpdate,
	}

	for k, v := range e.concentrationState.MessageCounts {
		state.MessageCounts[k] = v
	}
	for k, v := range e.concentrationState.Concentrations {
		state.Concentrations[k] = v
	}
	for k, v := range e.concentrationState.GradientVectors {
		state.GradientVectors[k] = v
	}

	return state
}

// GetReactionHistory returns recent chemical reactions
func (e *Engine) GetReactionHistory() []*types.ChemicalReaction {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Return a copy
	history := make([]*types.ChemicalReaction, len(e.reactionHistory))
	copy(history, e.reactionHistory)
	return history
}

// GetChemistryStats returns chemistry engine statistics
func (e *Engine) GetChemistryStats() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return map[string]interface{}{
		"node_id":             e.nodeID,
		"total_messages":      e.concentrationState.TotalMessages,
		"message_types":       len(e.concentrationState.Concentrations),
		"total_reactions":     len(e.reactionHistory),
		"concentration_decay": e.concentrationDecay,
		"diffusion_threshold": e.diffusionThreshold,
		"reaction_threshold":  e.reactionThreshold,
		"base_reaction_rate":  e.baseReactionRate,
		"last_update":         e.concentrationState.LastUpdate,
	}
}
