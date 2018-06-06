package gov

import (
	"fmt"
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Called every block, process inflation, update validator set
func EndBlocker(ctx sdk.Context, keeper Keeper) {

	// Delete proposals that haven't met minDeposit

	for shouldPopInactiveProposalQueue(ctx, keeper) {
		inactiveProposal := keeper.InactiveProposalQueuePop(ctx)
		keeper.DeleteProposal(ctx, inactiveProposal)
	}

	// Check if earliest Active Proposal ended voting period yet

	for shouldPopActiveProposalQueue(ctx, keeper) {
		activeProposal := keeper.ActiveProposalQueuePop(ctx)

		if ctx.BlockHeight() >= activeProposal.VotingStartBlock+keeper.GetVotingProcedure(ctx).VotingPeriod {
			passes, _ := tally(ctx, keeper, activeProposal)
			if passes {
				keeper.RefundDeposits(ctx, activeProposal.ProposalID)
			}
			keeper.DeleteProposal(ctx, activeProposal)
		}
	}

	return
}

func tally(ctx sdk.Context, keeper Keeper, proposal *Proposal) (passes bool, nonVoting []sdk.Address) {

	results := make(map[string]sdk.Rat)
	results["Yes"] = sdk.ZeroRat()
	results["Abstain"] = sdk.ZeroRat()
	results["No"] = sdk.ZeroRat()
	results["NoWithVeto"] = sdk.ZeroRat()

	totalVotingPower := sdk.ZeroRat()
	currValidators := make(map[string]validatorGovInfo)
	for _, val := range keeper.sk.GetValidatorsBonded(ctx) {
		currValidators[addressToString(val.Owner)] = validatorGovInfo{
			ValidatorInfo: val,
			Minus:         sdk.ZeroRat(),
		}
	}

	votesIterator := keeper.GetVotes(ctx, proposal.ProposalID)
	for ; votesIterator.Valid(); votesIterator.Next() {
		vote := &Vote{}
		keeper.cdc.MustUnmarshalBinary(votesIterator.Value(), vote)

		if val, ok := currValidators[addressToString(vote.Voter)]; ok {
			val.Vote = vote.Option
		} else {
			for _, delegation := range keeper.sk.GetDelegations(ctx, vote.Voter, math.MaxInt16) {
				val := currValidators[addressToString(delegation.ValidatorAddr)]
				val.Minus = val.Minus.Add(delegation.Shares)

				votingPower := val.ValidatorInfo.PoolShares.Amount.Mul(delegation.Shares)
				results[vote.Option] = results[vote.Option].Add(votingPower)
				totalVotingPower = totalVotingPower.Add(votingPower)
			}
		}
	}
	votesIterator.Close()

	nonVoting = []sdk.Address{}
	for _, val := range currValidators {
		if len(val.Vote) == 0 {
			nonVoting = append(nonVoting, val.ValidatorInfo.Owner)
		} else {
			sharesAfterMinus := val.ValidatorInfo.DelegatorShares.Sub(val.Minus)
			votingPower := sharesAfterMinus.Mul(val.ValidatorInfo.PoolShares.Amount)

			results[val.Vote] = results[val.Vote].Add(votingPower)
			totalVotingPower = totalVotingPower.Add(votingPower)
		}
	}

	tallyingProcedure := keeper.GetTallyingProcedure(ctx)

	if results["NoWithVeto"].Quo(totalVotingPower).GT(tallyingProcedure.Veto) {
		return false, nonVoting
	} else if results["Yes"].Quo(totalVotingPower.Sub(results["Abstain"])).GT(tallyingProcedure.Threshold) {
		return true, nonVoting
	} else {
		return false, nonVoting
	}
}

func shouldPopInactiveProposalQueue(ctx sdk.Context, keeper Keeper) bool {
	depositProcedure := keeper.GetDepositProcedure(ctx)
	peekProposal := keeper.InactiveProposalQueuePeek(ctx)

	if peekProposal.isActive() {
		return true
	} else if peekProposal.SubmitBlock+depositProcedure.MaxDepositPeriod >= ctx.BlockHeight() {
		return true
	}
	return false
}

func shouldPopActiveProposalQueue(ctx sdk.Context, keeper Keeper) bool {
	votingProcedure := keeper.GetVotingProcedure(ctx)
	peekProposal := keeper.ActiveProposalQueuePeek(ctx)

	if peekProposal.VotingStartBlock+votingProcedure.VotingPeriod >= ctx.BlockHeight() {
		return true
	}
	return false
}

func addressToString(addr sdk.Address) string {
	return fmt.Sprintf("%s", addr.Bytes())
}