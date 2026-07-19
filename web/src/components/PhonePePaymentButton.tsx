"use client";

import { Button } from "./ui/button";
import { PaymentEventSessionCreatedData } from "../contracts";
import { formatMoney } from "../utils/money";

interface PhonePePaymentButtonProps {
  paymentSession: PaymentEventSessionCreatedData;
}

export function PhonePePaymentButton({ paymentSession }: PhonePePaymentButtonProps) {
  const handlePay = () => {
    if (paymentSession.checkoutURL) {
      window.location.href = paymentSession.checkoutURL;
      return;
    }
    window.location.href = "/?payment=success";
  };

  return (
    <Button onClick={handlePay} className="w-full bg-[#5f259f] hover:bg-[#4a1d7a]">
      Pay {formatMoney(paymentSession.amount, paymentSession.currency)} with PhonePe
    </Button>
  );
}
