import { Context, NarrowedContext } from 'telegraf';
import { CallbackQuery, Message, Update } from '@telegraf/types';

export interface BotSessionData {
  selectedState?: string;
}

export interface BotContext extends Context {
  session: BotSessionData;
}

export type LocationContext = NarrowedContext<
  BotContext,
  Update.MessageUpdate<Message.LocationMessage>
>;

export type TextContext = NarrowedContext<
  BotContext,
  Update.MessageUpdate<Message.TextMessage>
>;

export type CallbackQueryContext = NarrowedContext<
  BotContext,
  Update.CallbackQueryUpdate<CallbackQuery.DataQuery>
>;
