/** Format fare amounts. Backend stores values in paise (1/100 INR). */
export function formatMoney(amount: number, currency = "INR"): string {
  const value = amount / 100;
  const code = currency.toUpperCase();
  if (code === "INR") {
    return `₹${value.toFixed(2)}`;
  }
  try {
    return new Intl.NumberFormat("en-IN", {
      style: "currency",
      currency: code,
    }).format(value);
  } catch {
    return `${code} ${value.toFixed(2)}`;
  }
}
