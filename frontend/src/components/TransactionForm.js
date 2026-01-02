import React, { useState } from "react";
import "./TransactionForm.css";

const TransactionForm = () => {
  const [formData, setFormData] = useState({
    from: "user1",
    to: "user2",
    amount: "100",
    txType: "swap",
  });
  const [submitting, setSubmitting] = useState(false);
  const [message, setMessage] = useState(null);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSubmitting(true);
    setMessage(null);

    try {
      const response = await fetch("http://localhost:8080/submit", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(formData),
      });

      const data = await response.json();

      if (response.ok) {
        setMessage({
          type: "success",
          text: `Transaction submitted! ID: ${data.transactionID.substring(
            0,
            8
          )}... (Priority: ${data.priority})`,
        });
      } else {
        setMessage({
          type: "error",
          text: `Error: ${data.message || "Failed to submit transaction"}`,
        });
      }
    } catch (error) {
      setMessage({
        type: "error",
        text: `Network error: ${error.message}`,
      });
    } finally {
      setSubmitting(false);
    }
  };

  const handleChange = (e) => {
    setFormData({
      ...formData,
      [e.target.name]: e.target.value,
    });
  };

  return (
    <div className="transaction-form-container">
      <h2>Submit Transaction</h2>
      <form onSubmit={handleSubmit} className="transaction-form">
        <div className="form-row">
          <div className="form-group">
            <label>From:</label>
            <input
              type="text"
              name="from"
              value={formData.from}
              onChange={handleChange}
              required
            />
          </div>

          <div className="form-group">
            <label>To:</label>
            <input
              type="text"
              name="to"
              value={formData.to}
              onChange={handleChange}
              required
            />
          </div>
        </div>

        <div className="form-row">
          <div className="form-group">
            <label>Amount:</label>
            <input
              type="text"
              name="amount"
              value={formData.amount}
              onChange={handleChange}
              required
            />
          </div>

          <div className="form-group">
            <label>Type:</label>
            <select
              name="txType"
              value={formData.txType}
              onChange={handleChange}
              required
            >
              <option value="swap">Swap (Priority 0 - Highest)</option>
              <option value="borrow">Borrow (Priority 1)</option>
              <option value="lend">Lend (Priority 2)</option>
              <option value="transfer">Transfer (Priority 3 - Lowest)</option>
            </select>
          </div>
        </div>

        <button type="submit" disabled={submitting} className="submit-button">
          {submitting ? "Submitting..." : "Submit Transaction"}
        </button>

        {message && (
          <div className={`message ${message.type}`}>{message.text}</div>
        )}
      </form>
    </div>
  );
};

export default TransactionForm;
