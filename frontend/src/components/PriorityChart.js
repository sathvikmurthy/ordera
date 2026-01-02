import React from "react";
import {
  PieChart,
  Pie,
  Cell,
  ResponsiveContainer,
  Legend,
  Tooltip,
} from "recharts";
import "./PriorityChart.css";

const PriorityChart = ({ mempoolStats }) => {
  const COLORS = {
    0: "#ff4444", // Swap - Red (highest priority)
    1: "#ff8800", // Borrow - Orange
    2: "#ffcc00", // Lend - Yellow
    3: "#44ff44", // Transfer - Green (lowest priority)
  };

  const priorityNames = {
    0: "Swap",
    1: "Borrow",
    2: "Lend",
    3: "Transfer",
  };

  const data = mempoolStats.byPriority
    ? Object.entries(mempoolStats.byPriority)
        .map(([priority, count]) => ({
          name: priorityNames[priority],
          value: count,
          priority: parseInt(priority),
        }))
        .filter((item) => item.value > 0)
    : [];

  if (data.length === 0) {
    return (
      <div className="priority-chart">
        <h2>📈 Priority Distribution</h2>
        <div className="no-data">No transactions in queue</div>
      </div>
    );
  }

  return (
    <div className="priority-chart">
      <h2>📈 Priority Distribution</h2>
      <ResponsiveContainer width="100%" height={300}>
        <PieChart>
          <Pie
            data={data}
            cx="50%"
            cy="50%"
            labelLine={false}
            label={({ name, percent }) =>
              `${name}: ${(percent * 100).toFixed(0)}%`
            }
            outerRadius={80}
            fill="#8884d8"
            dataKey="value"
          >
            {data.map((entry, index) => (
              <Cell key={`cell-${index}`} fill={COLORS[entry.priority]} />
            ))}
          </Pie>
          <Tooltip />
          <Legend />
        </PieChart>
      </ResponsiveContainer>
    </div>
  );
};

export default PriorityChart;
