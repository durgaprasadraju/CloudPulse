import { PaymentEventSessionCreatedData } from "../contracts"
import { Button } from "./ui/button"
import { loadStripe } from "@stripe/stripe-js"
import { formatMoney } from "../utils/money"

interface StripePaymentButtonProps {
  paymentSession: PaymentEventSessionCreatedData
  isLoading?: boolean
}

function isLocalPaymentSession(sessionID: string) {
  return sessionID.startsWith("cs_test_local_")
}

// Initialize Stripe only when a real publishable key is present
const publishableKey = process.env.NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY
const stripePromise =
  publishableKey && !publishableKey.includes("replace_me")
    ? loadStripe(publishableKey)
    : null

export const StripePaymentButton = ({
  paymentSession,
  isLoading = false,
}: StripePaymentButtonProps) => {
  const handlePayment = async () => {
    // Local prototype: skip Stripe Checkout and mark payment successful
    if (isLocalPaymentSession(paymentSession.sessionID)) {
      window.location.href = "/?payment=success"
      return
    }

    if (!stripePromise) {
      console.error("Stripe publishable key is not configured")
      return
    }

    const stripe = await stripePromise

    if (!stripe) {
      console.error("Stripe failed to load")
      return
    }

    const { error } = await stripe.redirectToCheckout({ sessionId: paymentSession.sessionID })

    if (error) {
      console.error("Payment error:", error)
    }
  }

  if (!isLocalPaymentSession(paymentSession.sessionID) && !publishableKey) {
    return (
      <Button
        disabled
        className="w-full bg-red-500 text-white"
      >
        Stripe API KEY is not set on the NEXTJS app
      </Button>
    )
  }

  return (
    <Button
      onClick={handlePayment}
      disabled={isLoading}
      className="w-full"
    >
      {isLoading
        ? "Loading..."
        : isLocalPaymentSession(paymentSession.sessionID)
          ? `Pay ${formatMoney(paymentSession.amount, paymentSession.currency)} (local)`
          : `Pay ${formatMoney(paymentSession.amount, paymentSession.currency)}`}
    </Button>
  )
}
