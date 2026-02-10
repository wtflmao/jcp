import React from 'react';
import { OrderBook as OrderBookType, formatPrice } from '../types';

interface OrderBookProps {
  data: OrderBookType;
  symbol?: string;
}

export const OrderBook: React.FC<OrderBookProps> = ({ data, symbol = '' }) => {
  // 安全检查：确保 data 及其属性存在
  const bids = data?.bids ?? [];
  const asks = data?.asks ?? [];

  // 计算委比：(委买量 - 委卖量) / (委买量 + 委卖量) * 100%
  const totalBidSize = bids.reduce((sum, b) => sum + b.size, 0);
  const totalAskSize = asks.reduce((sum, a) => sum + a.size, 0);
  const totalSize = totalBidSize + totalAskSize;

  const weibi = totalSize > 0
    ? ((totalBidSize - totalAskSize) / totalSize * 100).toFixed(2)
    : '0.00';
  const weibiBuy = totalSize > 0 ? (totalBidSize / totalSize * 100).toFixed(1) : '0';
  const weibiSell = totalSize > 0 ? (totalAskSize / totalSize * 100).toFixed(1) : '0';

  return (
    <div className="h-full flex flex-row fin-panel border-l fin-divider overflow-hidden text-xs font-mono select-none">
       {/* 买盘 */}
       <div className="flex-1 flex flex-col border-r fin-divider">
          <div className="p-2 border-b fin-divider font-bold text-slate-400 flex justify-between fin-panel-strong">
             <span>买盘</span>
             <span className="text-[10px] font-normal opacity-70">数量</span>
          </div>
          <div className="flex-1 overflow-hidden">
             {bids.slice(0, 15).map((bid, i) => (
                <div key={`bid-${i}`} className="relative flex justify-between px-2 py-0.5 hover:bg-slate-800/50 cursor-crosshair">
                   <div 
                    className="absolute top-0 left-0 bottom-0 bg-green-900/20 transition-all duration-300" 
                    style={{ width: `${Math.min(bid.percent * 5, 100)}%` }}
                  />
                  <span className="text-green-400 relative z-10">{formatPrice(bid.price, symbol)}</span>
                  <span className="text-slate-300 relative z-10">{bid.size}</span>
                </div>
             ))}
          </div>
       </div>

       {/* 委比信息 */}
       <div className="w-24 flex flex-col items-center justify-center border-r fin-divider fin-panel-strong z-10 shadow-inner">
           <div className="text-slate-500 text-[10px]">委比</div>
           <div className={`font-bold my-1 ${parseFloat(weibi) >= 0 ? 'text-red-400' : 'text-green-400'}`}>{weibi}%</div>
           <div className="text-[10px] text-slate-500">
             <span className="text-red-400">{weibiBuy}%</span>
             <span className="mx-1">/</span>
             <span className="text-green-400">{weibiSell}%</span>
           </div>
       </div>

       {/* 卖盘 */}
       <div className="flex-1 flex flex-col">
          <div className="p-2 border-b fin-divider font-bold text-slate-400 flex justify-between fin-panel-strong">
             <span>卖盘</span>
             <span className="text-[10px] font-normal opacity-70">数量</span>
          </div>
          <div className="flex-1 overflow-hidden">
            {asks.slice(0, 15).map((ask, i) => (
                <div key={`ask-${i}`} className="relative flex justify-between px-2 py-0.5 hover:bg-slate-800/50 cursor-crosshair">
                   <div 
                    className="absolute top-0 right-0 bottom-0 bg-red-900/20 transition-all duration-300" 
                    style={{ width: `${Math.min(ask.percent * 5, 100)}%` }} 
                  />
                  <span className="text-red-400 relative z-10">{formatPrice(ask.price, symbol)}</span>
                  <span className="text-slate-300 relative z-10">{ask.size}</span>
                </div>
            ))}
          </div>
       </div>
    </div>
  );
};
