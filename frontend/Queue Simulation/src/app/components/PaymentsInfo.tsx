import { CheckCircle, XCircle, IndianRupee } from "lucide-react";

interface Transaction {
  user: string;
  amount: string;
  status: "Success" | "Failed";
  date: string;
}

const transactions: Transaction[] = [
  { user: "Arjun", amount: "₹500", status: "Success", date: "2026-04-02" },
  { user: "Priya", amount: "₹700", status: "Success", date: "2026-04-02" },
  { user: "Rahul", amount: "₹400", status: "Failed", date: "2026-04-02" },
  { user: "Sneha", amount: "₹850", status: "Success", date: "2026-04-01" },
  { user: "Vikram", amount: "₹600", status: "Success", date: "2026-04-01" },
  { user: "Anika", amount: "₹550", status: "Failed", date: "2026-04-01" },
  { user: "Karan", amount: "₹900", status: "Success", date: "2026-03-31" },
  { user: "Meera", amount: "₹750", status: "Success", date: "2026-03-31" },
];

export function PaymentsInfo() {
  const successfulTransactions = transactions.filter((t) => t.status === "Success");
  const failedTransactions = transactions.filter((t) => t.status === "Failed");
  const totalRevenue = successfulTransactions.reduce((sum, t) => {
    const amount = parseInt(t.amount.replace(/[₹,]/g, ""));
    return sum + amount;
  }, 0);

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold text-gray-900 mb-2">Payment Transactions</h2>
        <p className="text-gray-600">View all booking transactions and their status</p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-green-50 p-4 rounded-lg border border-green-200">
          <div className="flex items-center gap-2 text-green-700 mb-2">
            <CheckCircle className="w-5 h-5" />
            <span className="text-sm font-medium">Successful</span>
          </div>
          <div className="text-2xl font-bold text-green-900">{successfulTransactions.length}</div>
        </div>
        <div className="bg-red-50 p-4 rounded-lg border border-red-200">
          <div className="flex items-center gap-2 text-red-700 mb-2">
            <XCircle className="w-5 h-5" />
            <span className="text-sm font-medium">Failed</span>
          </div>
          <div className="text-2xl font-bold text-red-900">{failedTransactions.length}</div>
        </div>
        <div className="bg-blue-50 p-4 rounded-lg border border-blue-200">
          <div className="flex items-center gap-2 text-blue-700 mb-2">
            <IndianRupee className="w-5 h-5" />
            <span className="text-sm font-medium">Total Revenue</span>
          </div>
          <div className="text-2xl font-bold text-blue-900">₹{totalRevenue.toLocaleString()}</div>
        </div>
      </div>

      {/* Transactions Table */}
      <div className="bg-white rounded-lg shadow overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-50 border-b border-gray-200">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  User
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Amount
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Status
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Date
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {transactions.map((transaction, index) => (
                <tr key={index} className="hover:bg-gray-50 transition-colors">
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="font-medium text-gray-900">{transaction.user}</div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="text-gray-900">{transaction.amount}</div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span
                      className={`inline-flex items-center gap-1 px-3 py-1 rounded-full text-xs font-medium ${
                        transaction.status === "Success"
                          ? "bg-green-100 text-green-800"
                          : "bg-red-100 text-red-800"
                      }`}
                    >
                      {transaction.status === "Success" ? (
                        <CheckCircle className="w-3 h-3" />
                      ) : (
                        <XCircle className="w-3 h-3" />
                      )}
                      {transaction.status}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-gray-500">
                    {transaction.date}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
